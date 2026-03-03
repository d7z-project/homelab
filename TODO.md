# IP 黑名单与合并导出策略功能设计方案 (IP Blacklist & Merging Policy)

## 1. 核心目标
构建一个企业级的单节点 IP 拦截与情报管理系统。支持动态封禁、多维度分组管理、基于 Cron 的自动同步，以及支持“正负组抵消”和“网段压缩”的高级合并导出策略。

---

## 2. 核心开发工作流 (Development Workflow)

遵循 `GEMINI.md` 规范，所有功能开发必须遵循以下生命周期：

### 第一阶段：分层后端开发 (Backend Implementation)
1.  **Models**: 定义 `IPBlacklistGroup`, `IPBlacklistEntry`, `IPBlacklistExportPolicy`。
    - **要求**：字段遵循 `camelCase`，所有请求 DTO 必须实现 `render.Binder`。
2.  **Repositories**: 实现基于 `kv.Child` 的持久化逻辑。
    - **要求**：使用 `Child("system", "ip_blacklist", ...)` 进行命名空间隔离。
3.  **Services**: 编排业务、执行校验（引入 `netip`）与权限过滤。
    - **审计规范**：所有增删改操作必须通过 `commonaudit.Log` 记录（含新旧值对比或删除快照）。
    - **细粒度权限**：调用 `commonauth.IsAllowed` 对具体 Group/Policy 实例进行精确拦截。
4.  **Controllers**: 挂载路由与中间件。
    - **路由准入**：挂载 `RequirePermission("admin", "ip_blacklist")`。

### 第二阶段：同步与验证 (Sync & Validate)
1.  **API 同步 (强制)**：执行 `make backend-generate` 同步 Swagger 文档及前端 `generated/` API 客户端。
2.  **后端测试**：执行 `go test ./tests/unit/...` 确保核心逻辑、权限拦截及审计覆盖率达标。

### 第三阶段：前端实现 (Frontend Implementation)
1.  **代码规范**：统一使用 Angular 17+ 现代控制流 (`@if`, `@for`)。
2.  **UI/UX**：遵循 Material Design 3 风格，使用 `requestAnimationFrame` 包装高频交互状态更新，避免 `NG0100` 错误。

---

## 3. 前端 UI/UX 设计 (Frontend Design)

前端统一入口位于侧边栏导航：**系统安全 (Security) -> IP 拦截 (IP Filter)**。页面采用 Angular Material 3 的 Tab 风格分为三个主视图：

### 3.1 视图 1：分组管理 (Groups)
- **卡片网格布局**：每个分组展示为一个独立的 Card 组件。
  - **常规组**：显示组名、IP/CIDR 记录数量、创建时间。提供“管理内容”、“重命名”、“删除”等标准操作。
  - **托管组 (Managed)**：Card 头部带 🔒 锁定图标。额外显示外部源 `SourceURL`、同步周期 `Cron` 以及最近一次同步状态（成功/失败时间）。操作区域仅提供“立即同步”和“配置策略”按钮，**隐藏所有直接修改内容的操作**。
- **分组创建/编辑向导**：弹窗表单支持切换分组类型。选择“外部同步”时，动态显示 URL 和 Cron 表达式的输入字段，并提供常用周期的快速选择（如：每天、每周）。

### 3.2 视图 2：黑名单条目 (Entries)
- **Master-Detail 布局**：
  - **左侧边栏**：提供可多选的分组过滤器（Checkbox List），支持按组快速筛选条目。
- **右侧主体数据表**：
  - **智能工具栏**：包含全局模糊搜索。若左侧过滤器选中了任何“托管组”，则自动禁用“添加条目”、“批量导入”等写操作按钮，以防止越权修改。
  - **扩展情报列 (GeoIP)**：数据表格除了显示 IP/CIDR、归属分组和封禁原因外，新增 **Geo情报列**。通过异步调用后端解析服务，展示该 IP 的国家/地区（带国旗 Emoji）及 ASN (Autonomous System Number) 信息。
  - **即时校验交互**：在“添加条目”弹窗中，利用 Angular 自定义 Validator 对输入的文本进行实时的 `IP / CIDR` 语法和逻辑校验（如检测无效子网掩码）。

### 3.3 视图 3：拦截策略 (Policies)
这是合并与导出逻辑的核心配置台：
- **策略列表**：展示已定义的合并策略，支持一键设置为“激活状态”（即作为全局拦截引擎的应用策略）或执行“全量导出”。
- **策略编辑器 (核心组件)**：
  - **双向选择穿梭框**：左右两栏分别配置 `IncludeGroups`（正向包含组，UI 主色调为绿色/蓝色）和 `ExcludeGroups`（负向排除组，UI 主色调为红色/橙色，寓意从中剔除）。
  - **流式预览控制台 (Live Console)**：当配置发生变更时，自动触发后台 `PreviewPolicy` 接口。以类似终端日志的黑底绿字面板，流式打印（NDJSON）合并后的网段列表。同时顶部给出数据统计卡片：**原始条目数 -> 合并重叠后条目数 -> 负向剔除后最终条目数**，直观展示 CIDR 压缩比。

### 3.4 全局交互：命中测试工具 (Hit Test Tool)
- **悬浮抽屉**：在页面右下角提供一个全局 FAB (Floating Action Button) 呼出“调试实验室”抽屉。
- **功能设计**：
  1.  **输入与情报 (Input & GeoInfo)**：管理员输入任意 IP 地址，抽屉顶部立刻通过 GeoIP2 呈现该 IP 的归属地、网络服务商及 ASN。
  2.  **命中路径推演 (Hit Path Trace)**：选择当前应用策略后点击测试，面板将以高亮链路形式展示判断逻辑：
      - *示例 A*：未命中任何正向组 -> 绿灯 `ALLOWED`。
      - *示例 B*：命中恶意扫描组 (包含) -> 未命中排除组 -> 红灯 `BLOCKED`。
      - *示例 C*：命中恶意扫描组 (包含) -> 命中自建白名单组 (排除) -> 抵消 -> 绿灯 `EXCLUDED (ALLOWED)`。

---

## 4. 技术细化 (Technical Specifications)

### 4.1 存储与同步引擎 (Storage & Sync)
- **KV 增强替换**：利用 BoltDB 的 `Child` 特性，在托管组全量同步时实现“原子全量替换”（即先写入 Temp 节点 -> 校验无误 -> 逻辑覆盖正式节点）。
- **定时调度**：集成本地 `robfig/cron/v3`，由单实例进程负责所有组的外部源拉取任务。
- **错误容忍**：拉取失败或解析错误时支持退避重试，并保留上一版本的有效数据。

### 4.2 匹配索引与合并运算 (Matching & Merging)
- **极速拦截 (Radix Tree)**：
  - 内存中维护基数树匹配索引。
  - 支持 **原子切换指针 (Atomic Pointer Swap)**，在后台树重构完成后瞬间切换，实现拦截引擎的零停机更新。
- **合并引擎 (netipx.IPSet)**：
  - 支持复杂策略运算：合并所有 `IncludeGroups` -> 剔除 `ExcludeGroups` 命中的网段（支持大网段内挖小洞）。
  - 执行 CIDR 自动压缩以精简输出。

### 4.3 GeoIP2 情报解析 (GeoIP2 Resolution)
- **数据库订阅与管理**：
  - 支持配置 GeoLite2 (Country/ASN) 数据库的下载 URL 和许可证密钥 (License Key)。
  - 利用现有的 Cron 引擎，定期（如每周）自动下载并热加载 `.mmdb` 格式的数据库文件，持久化存储在特定的数据目录。
- **IP 信息查询服务**：
  - 集成 `github.com/oschwald/geoip2-golang` 库加载内存映射数据库。
  - 提供内部服务 API，输入任意 IP 或 CIDR，返回其关联的 ASN、国家代码及地理位置信息。
  - **性能优化**：采用 `mmap` 方式加载以减少内存占用，并在文件更新时通过读写锁安全切换读取器 (Reader)。

---

## 5. 任务清单 (Action Plan)

- [ ] **Task 1: [Backend/Models]** 定义核心模型（Group/Entry/Policy）并实现 `render.Binder` 接口。
- [ ] **Task 2: [Backend/Repo]** 实现 KV 分组隔离存储与托管组的原子刷新（Temp-Swap）机制。
- [ ] **Task 3: [Backend/Service]** 开发 `netip` 校验逻辑、正负组抵消算法（`netipx`）以及审计/权限安全拦截。
- [ ] **Task 4: [Backend/Middleware]** 开发并挂载基于内存 Radix Tree 的高性能 IP 全局拦截中间件。
- [ ] **Task 5: [Backend/GeoIP2]** 集成 `geoip2-golang`，实现数据库的定时下载管理与 IP 情报解析服务。
- [ ] **Task 6: [Frontend]** 开发策略配置中心 UI，支持可视化分组编排、GeoIP 信息展示、实时效果预览与 Cron 订阅配置。
- [ ] **Task 7: [Full-Stack]** 运行 `make all`，验证前后端全栈编译及端到端功能。
