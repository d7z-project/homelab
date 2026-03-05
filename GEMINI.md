# Gemini 项目上下文

## 核心架构决策：分层开发范式 (Layered Architecture)

为了保证代码的可维护性与高性能，本项目严格遵循以下四层分层架构。新功能的开发必须按此顺序进行：

### 1. 模型层 (Models) - `pkg/models/`
- **职责**：定义所有业务实体、API 请求/响应 DTO，并承担 **基础格式校验** 职责。
- **规范**：
  - 所有结构体字段必须带有 camelCase 格式标签。
  - **校验闭环 (Mandatory)**：必须实现 `render.Binder` 接口。
  - **Bind 职责**：负责所有非空检查、正则表达式校验（如 ID 格式 `^[a-z0-9_]+$`）、枚举范围检查及字段预处理（如 TrimSpace）。
  - **双层正则校验**：涉及动态参数 or 变量定义时，应提供 `RegexFrontend` (UI 实时反馈) 与 `RegexBackend` (执行前强制校验)。若输入包含 `${{ ... }}` 变量语法，前端校验应自动跳过以允许动态注入。

### 2. 存储仓库层 (Repositories) - `pkg/repositories/{module}/`
- **职责**：封装底层的 KV 存取逻辑及缓存策略。
- **子命名空间规范**：利用 `common.DB.Child(namespace...)` 进行资源隔离（如 `system`, `audit`）。

### 3. 业务服务层 (Services) - `pkg/services/{module}/`
- **职责**：编排业务流程、执行 **深度业务逻辑校验**、维护资源一致性。
- **校验规范**：Service 层方法在执行任何逻辑前，**必须显式调用** `model.Bind(nil)`。
- **更新一致性 (Critical)**：在执行 `Update*` 操作时，必须显式执行 `model.ID = id`（将 URL 路径中的 ID 强制覆盖至 Body 结构体），以防止因 ID 缺失或不一致导致产生冗余记录。
- **引用合法性校验 (Mandatory)**：在创建或更新包含外键引用（如 `ServiceAccountID`, `RoleIDs`, `DomainID`）的资源时，Service 层必须在持久化前 **强制校验** 被引用资源的存在性。
- **删除完整性 (Cascading & Protection)**：
  - **级联删除**：父级资源（如 `Domain`, `Workflow`）被删除时，其所属的子资源（如 `Record`, `TaskInstance`）必须通过 Repository 同步清理。
  - **引用保护**：被其他核心模块引用的基础身份资源（如正被工作流引用的 `ServiceAccount`），在解除引用关系前 **严禁删除**。
- **文件操作规范 (VFS)**：所有涉及业务数据及临时空间的读写必须分别通过 `common.FS` 或 `common.TempDir` (均为 `afero.Fs`) 执行。严禁直接使用 `os` 包或进行物理路径拼接。
- **模块级沙箱 (Scoped FS)**：子模块若需管理子目录，必须基于全局 Fs 通过 `afero.NewBasePathFs` 建立包级私有 Fs（如 `orchFS`）。业务逻辑应直接面向私有 Fs 的根路径操作。
- **任务执行上下文 (`TaskContext`)**：其 `Workspace` 必须为 `afero.Fs` 类型，且必须通过 `afero.NewBasePathFs` 严格限制在任务私有目录内。
- **日志操作规范**: 
  - **分片存储**: 任务日志必须按步骤索引分片存储（`{id}.{index}.log`），其中 `0` 为初始化，`N+1` 为清理。**分片日志中严禁输出冗余的步骤启停文本，仅记录处理器产生的核心信息。**
  - **增量查询**: 必须支持基于行偏移量 (`offset`) 的增量拉取，以优化实时日志性能。
  - **耗时追踪**: 必须实时维护 `StepTimings` 记录每一分片的 `StartedAt` 与 `FinishedAt`。
  - **状态同步**: 步骤切换时必须立即持久化 `CurrentStep`，确保前端 UI 能够精准同步执行进度。
  - **自动清理**: 任务实例或父级工作流删除时，必须级联清理所有分片日志。
- **智能资源清理**: 执行自愈或清理逻辑前，必须核查数据库状态。仅当关联任务处于终态（Failed/Success）或记录不存在时，才允许物理/逻辑删除对应的虚拟目录。
- **资源清理规范**：所有任务必须在 `run` 函数中使用 `defer` 确保在任何退出场景下执行逻辑清理。
- **审计规范**：所有修改类操作（C/U/D）及触发类操作（Trigger）必须手动上报审计日志。

---

## 核心基础设施 (Core Infrastructure)

项目采用高度虚拟化的基础设施层，通过 URL Scheme 驱动实现“默认内存、按需物理”的部署策略。

### 1. 资源初始化模式
- **存储 (DB)**: `common.DB` (kv.KV)，支持 `memory://`, `etcd://`, `redis://`。
- **锁 (Locker)**: `common.Locker` (lock.Locker)，支持 `memory://`, `etcd://`。
- **虚拟文件系统 (VFS)**: 
  - **业务存储 (`common.FS`)**: 默认 `memory://`。
  - **临时空间 (`common.TempDir`)**: 默认 `memory://`。
  - **初始化安全**: `common.InitVFS` 强制使用 `BasePathFs` 封装。`local://` 模式下严禁隐式回退，必须提供显式路径。
  - **启动自检**: 系统启动时必须对上述两个 Fs 执行“随机文件写入擦除”冒烟测试。

### 2. 模块初始化协议
- **Init() 模式**: 具备私有沙箱的模块（如 `orchestration`）必须提供 `Init()` 函数，在 `main` 建立全局 Fs 后同步完成包级 Scoped FS 的锚定。

### 3. 生命周期与停机
- **Context 驱动**: `main.go` 采用 `signal.NotifyContext` 建立全局根上下文。
- **优雅停机**: 所有初始化函数必须响应 `ctx.Done()`；`http.Server` 停机超时统一设为 5s。

遵循 **Material Design 3 (M3)** 交互规范，追求极致的视觉一致性与操作流畅度。

### 1. 页面结构标准
- **标准化页头**：所有列表类页面必须使用 `PageHeaderComponent` (`app-page-header`)。
  - 统一显示大标题、功能说明、数据总计及标准的“刷新”按钮。
  - 搜索状态、过滤组件应通过 `chips` 插槽插入页头。
- **任务编排交互**:
  - **步骤排序**: 步骤配置必须支持 `cdkDrag` 拖拽排序。
  - **处理器选择**: 必须通过具备搜索功能的 `Dialog` 进行选择，禁止使用过长的下拉菜单。
- **底部边距**：页面根容器必须包含 `pb-20` 类，以确保在不同设备上拥有统一的底部呼吸空间。

### 2. 表格展示规范
- **状态切换 (Status Toggle)**：
  - 所有带有“启用/禁用”功能的表格，其切换开关必须位于 **第一列**。
  - 统一使用 `mat-slide-toggle` 组件，并应用 `scale-75` 样式。
- **实时性要求**：
  - **列表页**: 状态频繁变更的监控页面（如“运行记录”）应实现 **10s 自动刷新**。
  - **详情页**: 实时日志及执行进度应实现 **2s 自动刷新**，支持自动滚动、步骤自动跟随及 **智能初始定位**（打开时自动跳转至执行中或失败的步骤）。
- **交互一致性**: 点击列表中的资源名称（如工作流名）应自动触发“按该资源过滤”并切换至关联的监控视图（如运行记录）。

### 3. 居中与布局规范 (Centering & Layout)
- **图标按钮居中**: 
  - 所有 `mat-icon-button` 必须确保图标在视觉上绝对居中。
  - 若手动设置按钮尺寸（如 `!w-8 !h-8`），必须应用 `icon-button-center` 类以强制 flex 居中。
  - 图标尺寸建议使用标准的 `!text-[20px]` 或 `!text-lg` 以保持与 Material Design 3 规范一致。

### 4. 变更检测与错误处理 (Angular NG0100)
- **异步更新规范**: 
  - **NG0100 防御**: 涉及复杂表单联动、异步建议加载或在 `ngAfterViewInit` 阶段更新 UI 状态时，**必须**将状态更新逻辑包裹在 `setTimeout` 或 `Promise.resolve().then()` 中。
  - **检测触发**: 更新异步状态后，应优先使用 `cdr.markForCheck()` 以允许 Angular 在下一轮周期自动同步；若涉及子组件验证状态的立即回传，可使用 `cdr.detectChanges()`。
  - **校验解耦**: 复杂的 `isValid()` 判断若依赖于 `@ViewChildren` 的验证状态，应通过信号 (Signal) 或专用变量在特定事件（如 `ngModelChange`）中更新，**禁止**在模板表达式中直接进行深度遍历校验以防检测冲突。

---

## 导航与菜单
- **分组策略**：系统级配置与监控功能（审计日志、会话管理等）应统一收纳于“系统管理”菜单组内。
- **权限隔离**：菜单项的显示应根据 `uiService.userType()` 进行实时权限过滤（如“管理会话”仅对 `root` 可见）。

---

## 测试框架与质量保证

### 1. 后端功能测试 - `backend/tests/unit/`
- **覆盖要求**：核心业务逻辑必须具备 100% 的功能测试覆盖。
- **关键路径验证**：必须包含针对 Update ID 一致性、物理路径清理逻辑及 RBAC 细粒度过滤的验证用例。
- **一致性与幂等性测试**: 必须在 `consistency_test.go` 中包含针对跨模块引用校验、删除保护以及高并发下 `TryLock` 拦截效果的专项验证。

---

## 核心开发工作流 (Core Workflow)

### 1. 迭代开发循环
1. **后端实现**：Models -> Repositories -> Services -> Controllers。
2. **逻辑验证**：执行 `go test ./tests/unit/...` 确保逻辑闭环。
3. **API 同步 (强制)**：运行 `make backend-gen`。
4. **全栈构建验证 (强制)**：交付前运行 `make all` 确保全栈类型定义无冲突。
