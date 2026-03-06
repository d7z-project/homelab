# 域名池与动态导出过滤引擎 (Site Pool & Dynamic Export Engine)

## 1. 业务目标 (Business Goal)
构建一个支持多标签 (Tag)、极速检索的域名池 (Site/Domain) 管理系统，并提供基于表达式 (`go-expr`) 的无状态动态导出能力。专为路由代理 (如 v2ray `geosite.dat`)、DNS 拦截 (如 AdGuardHome / DNSmasq) 等场景设计。
**核心原则：DB 存元数据，VFS 存池数据，缓存驱动导出，多模式树 (Trie/AC/Regex) 支撑查询。**

---

## 2. 核心架构设计 (Core Architecture Design)

严格遵循 `GEMINI.md` 规定的四层分层架构与基础设施规范。

### 2.1 域名池管理 (Site Pool Management)
- **元数据 (DB)**: 存储 `SiteGroup` 的基本信息（UUID、名称、描述、来源类型、数据指纹 Checksum 等）。
    - **隔离规范**: Repository 层必须使用 `common.DB.Child("network", "site")` 进行命名空间隔离。
- **实体数据 (VFS)**: 实际的域名规则（`domain:`, `full:`, `keyword:`, `regexp:`）存储于 `vfs://network/site/pools/{uuid}.bin`（基于 `common.FS` 封装）。
    - **安全删除**: 级联删除机制。若存在 `SiteExport` 引用则**拒绝删除**；允许删除时同步删除 VFS `.bin` 文件。
- **多维度标签 (Tags)**: 每一条规则可附加 Tag。通常一个 `geosite.dat` 包含多个 Tag（如 `cn`, `google`, `apple`），系统需无损保留并支持基于 Tag 的拆分组合。
- **规则规整化 (Normalization)**: 导入时强制转小写、去除不可见字符、Punycode 转换等清洗操作。
- **容量与雪崩防护 (Capacity & Anti-Avalanche)**:
    - 针对海量域名的分段流式载入 (`io.Reader`)，严禁 `ReadAll` 防 OOM。
- **数据预览 (Preview)**: 采用 **游标分页 (Cursor-based)** 替代 Offset/Limit，实现 O(1) 性能的流式数据展示。

### 2.2 动态过滤与多格式导出 (Dynamic Export)
- **无状态持久化**: 导出配置 (`SiteExport`) 仅在 DB 中存储过滤规则，不生成永久物理文件。
- **多格式支持 (Export Formats)**:
    - `application/vnd.v2ray.geosite`: 导出为标准的 `geosite.dat` 二进制格式，供 v2ray/Xray 等直接挂载。
    - `text/plain`: 换行符分隔的纯文本格式（可指定输出模式为单纯的 Domain 列表，或带类型的 `full:xxx` 列表）。
    - `application/x-adguard-filter`: 转换为 AdGuard Home 兼容的 DNS 拦截规则格式 (`||example.com^`)。
    - `application/x-dnsmasq`: 转换为 DNSmasq 配置格式 (`server=/example.com/127.0.0.1`)。
- **异步任务驱动 (Async Task Model)**:
    - **API 拆分**: 独立的“触发生成”与“获取下载链接”接口。包含 `float` 进度返回与缓存秒级响应。
    - **抢占与最新保留 (Cancel & Replace)**: 同一个配置的并发触发会自动取消旧任务，仅保留最新。
- **域名规则合并压缩 (Domain Deduplication)**:
    - 导出器在执行集合并运算时，需要执行子域消除逻辑。例如：若同时包含 `domain:google.com` 与 `full:www.google.com`，则后者将被作为冗余项合并消除。
- **带有 TTL 的依赖感知缓存**:
    - 缓存于 `common.TempDir` 下，Cache Key 由 “规则哈希 + 源池 Checksum + 目标格式” 决定。预留 `TriggerGC()` 接口。

### 2.3 辅助研判能力 (Analysis Utilities)
- **并发安全的全局缓存**: 使用 `atomic.Value` / `sync.RWMutex` 维护的全局内存匹配引擎池。
- **命中推演 (Hit Test)**: 提供沙盒 API，输入一个测试域名 (如 `www.google.com`)，极速研判其是否命中指定池，并返回具体命中的规则（是匹配了 `keyword:goog` 还是 `domain:google.com`）及其关联的 Tags。

### 2.4 全局审计与监控 (Audit & Observability)
- 强制接入 `commonaudit.Log` 记录核心操作。

---

## 3. 数据结构与内存模型 (Data & Memory Specs)

### 3.1 VFS 二进制格式 (Tagged Site Binary Format)
针对 v2ray 路由规则复杂度设计的自定义或对齐开源标准的格式：
- **Header**: `Magic` | `Version` | `EntryCount` | `Checksum` | `DictOffset`。
- **Dictionary Block (常量字符串字典)**: 去重存储所有 Tag。
- **Payload**: `[RuleType (1 byte)]` + `[Value (字符串)]` + `[Tags 索引数组]`。
  *(RuleType: 0=Plain/Keyword, 1=Regex, 2=Domain, 3=Full)*

### 3.2 内存匹配引擎 (Domain Matching Engine)
由于域名匹配涉及多种模式，无法使用单一树结构。必须实现复合型的匹配器 (Composite Matcher)：
- **Domain Trie (域名后缀树)**: 处理 `domain:` 和 `full:` 规则。通过域名层级倒序构建 Trie 树（如 `com -> google -> www`），实现 O(L) 的精准匹配。
- **Aho-Corasick 自动机**: 处理 `keyword:` 规则，实现多模式字符串极速并发搜索。
- **Regex Pool**: 处理 `regexp:` 规则，对正则进行预编译并缓存。
- **Value 载体**: 匹配成功后，必须能返回附加的 **Tag 集合**。

---

## 4. 处理器规格 (Processor Specifications)

### 4.1 处理器：`site/pool/import`
- **逻辑**: 解析外部列表，执行规整化与去重合并，并执行 `Temp-and-Rename` 覆盖至 `vfs://network/site/pools/{group_uuid}.bin`。
- **并发与锁调度**: **全局任务串行 (Strict Serialization)**，防止内存与 CPU 毛刺。
- **输入参数 (Action Inputs)**:
    - `filePath`: 待导入的本地文件路径。
    - `format`: `geosite` (v2ray标准dat), `text` (每行一个域名), `adguard` 等。
    - `mode`: `append` 或 `delete`。
    - `targetPool`: 目标 `group_uuid`。
    - `defaultTags`: 需要附加的 Tag，或者在 delete 模式下作为精确消除条件。

---

## 5. 模块组成与架构合规性 (System Modules & Compliance)

### 5.1 API 路由规范 (API Endpoint Paths)
- **域名池管理 (Site Pools)**: `/api/v1/network/site/pools`
- **命中推演与查询 (Analysis)**: `/api/v1/network/site/analysis`
- **动态导出 (Exports)**: `/api/v1/network/site/exports`

### 5.2 架构合规规范
- **模型层**: 强制实现 `render.Binder` 校验。
- **服务层**: 权限错误采用 `%w` 包装 `commonauth.ErrPermissionDenied`，向 Discovery 注册服务发现时必须实现分页除零安全。
- **控制器层**: 链式调用 `r.With(middlewares.RequirePermission(...))`，错误全权交由 `controllers.HandleError` 探测与映射。

---

## 6. 任务清单 (Action Plan)

严格遵循 `GEMINI.md` 核心开发工作流。

### 第一阶段：基础设施 (Infrastructure & Storage)
- [ ] **Task 1: [Models]** 定义 `SiteGroup` 与 `SiteExport` 模型。实现 `render.Binder` 验证。
- [ ] **Task 2: [Tagged-Codec]** 实现支持 Tag 与 4 种路由规则的二进制编解码器 (`.bin` / `geosite.dat` 解析)。**要求流式读取**。
- [ ] **Task 3: [Matching-Engine]** 开发基于 Domain Trie 和 Aho-Corasick 的内存复合匹配引擎，支持 Tag Payload 返回。

### 第二阶段：核心业务逻辑 (Core Services & Repositories)
- [ ] **Task 4: [Repositories]** 基于 `common.DB.Child("network", "site")` 实现 DB 层持久化。
- [ ] **Task 5: [Services]** 实现游标数据预览接口、安全级联删除逻辑，接入 `commonaudit.Log`，并完成 Discovery `LookupFunc` 注册。
- [ ] **Task 6: [Analysis-Engine]** 结合 `atomic.Value` 全局缓存，开发沙盒命中推演 (Hit Test) API。
- [ ] **Task 7: [Action-Handlers]** 实现 `site/pool/import` 处理器，确保全局队列严格串行调度。
- [ ] **Task 8: [Controllers]** 挂载链式路由与统一异常处理。

### 第三阶段：动态导出与逻辑合并 (Dynamic Export)
- [ ] **Task 9: [Expr-Engine]** 集成 `go-expr` AST 预分析降级，实现 Domain 重复规则的合并压缩 (Deduplication Algorithm)。
- [ ] **Task 10: [Formatters]** 实现多目标渲染器：`geosite.dat`, `text/plain`, `adguard`, `dnsmasq`。
- [ ] **Task 11: [Export-Cache]** 实现基于异步任务驱动、依赖感知 (Checksum) 且支持 TTL 回收的本地导出系统。
- [ ] **Task 12: [Tests & Sync]** 编写包含海量子域名聚合的性能单测与安全实例级隔离测试，并执行 `make backend-gen`。

### 第四阶段：前端与交付 (Frontend Adapters)
- [ ] **Task 13: [UI-Pools]** M3 规范开发域名池管理界面（含流式预览与无权限提示）。
- [ ] **Task 14: [UI-Analysis]** 开发域名规则“命中推演实验室”面板。
- [ ] **Task 15: [UI-Export]** 开发异步导出配置与进度面板，妥善处理缓存过期失效的 UX 展示。
- [ ] **Task 16: [Build]** 运行 `make all` 确保全栈编译通过。
