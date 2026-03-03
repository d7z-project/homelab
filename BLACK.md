# IP 黑名单与合并导出策略 (IP Blacklist & Merging Policy)

## 1. 业务目标 (Business Goal)
构建高性能、原子化更新的 IP 情报拦截系统。**所有外部情报的自动化获取与解析由通用任务编排引擎驱动。**

---

## 2. 处理器规格 (Processor Specifications)

### 2.1 处理器：`ip/parse/geoip-dat`
- **逻辑**: 读取二进制 `geoip.dat`。根据 Tag 过滤并将解析出的 IP 写入工作空间临时文件。
- **审计**: 通过 `commonaudit` 记录解析情况（文件名、匹配 Tags、条目数）。

### 2.2 处理器：`ip/load/group`
- **逻辑**: 执行 **全量原子替换 (Temp-Swap)**。读取文件内容并构建新 Radix Tree 索引。
- **并发冲突控制**: **模块内部自理。** 在执行 Load 期间，对目标 `group_id` 使用 `sync.Mutex` 或 KV 级排他锁，后续竞争冲突直接返回 `error` 任务失败。
- **实时权限校验**: 在 `Execute` 内部调用 RBAC 服务，根据 `TaskContext.UserID` 校验是否具备该 IP 分组的写权限。
- **状态维护**: 加载完成后，将分组标记为托管状态，锁定 UI 手动编辑功能。
- **审计**: 记录执行结果摘要（新增/删除条目数）至系统全局审计日志。

---

## 3. 技术实施规格 (Technical Specifications)

### 3.1 极速拦截中间件
- **基数树 (Radix Tree)**: 内存中维护全量 IP 索引。支持 O(K) 极速匹配（K 为 IP 位深）。
- **原子热更新**: 构建新树并完成预热后，原子级切换全局指针，确保在黑名单百万级更新时拦截不中断、零抖动。

### 3.2 策略引擎
- **合并运算**: 使用 `netipx.IPSet` 实现包含与排除逻辑的精准抵消。
- **导出压缩**: 自动将散乱网段压缩为最小 CIDR 集合。

---

## 4. 任务清单 (Action Plan)

严格遵循 `GEMINI.md` 核心开发工作流规范。

### 第一阶段：后端分层实现 (Backend Implementation)
- [ ] **Task 1: [Models]** 定义 `IPBlacklistGroup`, `IPBlacklistEntry`, `IPBlacklistExportPolicy` 模型，确保请求 DTO 实现 `render.Binder`。
- [ ] **Task 2: [Repositories]** 实现基于 `kv.Child("system", "ip_blacklist")` 的持久化逻辑，以及用于 `ip/load/group` 的并发排他锁。
- [ ] **Task 3: [Services]** 
  - 实现 IP 分组与条目 CRUD 业务，严格调用 `commonaudit.Log` 和 `commonauth.IsAllowed`。
  - 实现正负组抵消的策略导出逻辑 (`netipx.IPSet`)。
  - 向通用编排引擎注册 `ip/parse/geoip-dat` 与 `ip/load/group` 处理器。
- [ ] **Task 4: [Controllers]** 挂载 IP 管理路由，并使用 `RequirePermission("admin", "ip_blacklist")` 中间件防御。实现基于 Radix Tree 的全量拦截中间件。

### 第二阶段：逻辑验证与 API 同步 (Verify & Sync)
- [ ] **Task 5: [Tests]** 编写涵盖并发安全、细粒度权限过滤、审计日志生成的单元测试。编写针对 Radix Tree 原子切换的高并发测试。
- [ ] **Task 6: [Generate]** 执行 `make backend-generate`，更新 Swagger 文档及前端 API 客户端。

### 第三阶段：前端实现 (Frontend Adaptation)
- [ ] **Task 7: [UI-Groups]** 开发分组管理界面（使用 Angular 17+ 现代控制流），根据托管状态标志全局锁定条目的写操作。
- [ ] **Task 8: [UI-Policy]** 开发合并策略导出配置中心（双向选择穿梭框）。
- [ ] **Task 9: [UI-Test]** 实现命中推演实验室 (Hit Test Tool)，调用后端接口验证 IP 拦截路径。

### 第四阶段：全栈交付验证 (Full-Stack Verification)
- [ ] **Task 10: [Build]** 运行 `make all`，验证前后端编译，确保 API 定义一致，无类型定义冲突。
