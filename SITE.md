# 域名池与动态导出过滤引擎 (Site Pool & Dynamic Export Engine)

## 1. 业务目标 (Business Goal)
构建一个支持多标签 (Tag)、极速检索的域名池 (Site/Domain) 管理 system，并提供基于表达式 (`go-expr`) 的无状态动态导出能力。专为路由代理 (如 v2ray `geosite.dat`)、DNS 拦截 (如 AdGuardHome / DNSmasq) 等场景设计。
**核心原则：DB 存元数据，VFS 存池数据，缓存驱动导出，多模式树 (Trie/AC/Regex) 支撑查询。**

---

## 2. 核心架构设计 (Core Architecture Design)

严格遵循 `GEMINI.md` 规定的四层分层架构与基础设施规范。

### 2.1 域名池管理 (Site Pool Management)
- **元数据 (DB)**: 存储 `SiteGroup` 的基本信息（UUID、名称、描述、数据指纹 Checksum 等）。
    - **ID 生成**: 标识 ID (UUID) 必须由后端在创建时自动生成，不允许用户自定义。
    - **隔离规范**: Repository 层必须使用 `common.DB.Child("network", "site")` 进行命名空间隔离。
- **实体数据 (VFS)**: 实际的域名规则（`domain:`, `full:`, `keyword:`, `regexp:`）存储于 `vfs://network/site/pools/{uuid}.bin`（基于 `common.FS` 封装）。
    - **安全删除**: 级联删除机制。若存在 `SiteExport` 引用则**拒绝删除**；允许删除时同步删除 VFS `.bin` 文件。
- **条目级管理 (Entry Management)**: 支持在特定域名池内直接新增、修改、删除独立的规则记录。
    - **禁止重复**: 必须校验 `[RuleType + Value]` 组合在当前池中是否已存在，防止重复添加。
    - **修改限制**: 已存在条目仅允许修改关联的 `Tags`，禁止修改 `Value` 或 `RuleType`。
- **多维度标签 (Tags)**: 每一条规则可附加 Tag。通常一个 `geosite.dat` 包含多个 Tag（如 `cn`, `google`, `apple`），系统需无损保留并支持基于 Tag 的拆分组合。
- **规则规整化 (Normalization)**: 导入与新增时强制执行以下操作：
    - **清洗**: 转小写、去除首尾空格、去除不可见字符。
    - **编码**: 强制执行 Punycode (IDNA) 转换，确保国际化域名（中文域名）的匹配兼容性。
- **容量与雪崩防护 (Capacity & Anti-Avalanche)**:
    - 针对海量域名的分段流式载入 (`io.Reader`)，严禁 `ReadAll` 防 OOM。
- **数据预览与检索 (Preview & Search)**: 采用 **游标分页 (Cursor-based)** 替代 Offset/Limit，实现 O(1) 性能的流式数据展示。支持基于 Value 或 Tag 的文本搜索过滤。

### 2.2 动态过滤与多格式导出 (Dynamic Export)
- **无状态持久化**: 导出配置 (`SiteExport`) 仅在 DB 中存储过滤规则，不生成永久物理文件。
- **域名规则合并压缩 (Subdomain Deduplication)**:
    - 导出器在执行集合并运算时，必须执行“子域消除逻辑”。
    - **逻辑规则**: 若池中已存在 `domain:google.com`，则任何属于该后缀的 `full:www.google.com` 或 `domain:mail.google.com` 均被视为冗余。
    - **标签合并 (Tag Merging)**: 在消除冗余规则前，必须将其关联的 Tags 合并至保留的父级规则中，确保分流策略不丢失。
    - **注意**: `keyword` 和 `regexp` 类型不参与此自动消除逻辑，保持原样导出。
- **多格式支持 (Export Formats)**:
    - `application/vnd.v2ray.geosite`: 导出为标准的 `geosite.dat` 二进制格式（Protobuf 序列化）。
    - `text/plain`: 换行符分隔。支持 `Simple` (仅域名) 和 `Verbose` (带类型前缀) 两种模式。
    - `application/x-adguard-filter`: 转换为 AdGuard Home 兼容格式 (`||example.com^`)。
    - `application/x-dnsmasq`: 转换为 DNSmasq 格式 (`server=/example.com/127.0.0.1`)。
- **异步任务驱动 (Async Task Model)**:
    - **抢占与最新保留 (Cancel & Replace)**: 同一个配置的并发触发会自动取消旧任务，仅保留最新。
- **带有 TTL 的依赖感知缓存**:
    - 缓存于 `common.TempDir` 下，Cache Key 由 “规则哈希 + 源池 Checksum + 目标格式” 决定。

### 2.3 辅助研判能力 (Analysis Utilities)
- **并发安全的全局缓存**: 使用 `atomic.Value` 维护的内存匹配引擎。
- **命中推演 (Hit Test)**: 提供沙盒 API，输入域名返回匹配详情：
    - **返回项**: `Matched (bool)`, `RuleType`, `Pattern (命中原形)`, `Tags (关联标签组)`。
    - **匹配策略**: 采取“全量命中”原则。若一个域名同时被 `keyword:goog` 和 `domain:google.com` 命中，接口应返回所有命中规则的 Tags 交集/并集（取决于配置）。

---

## 3. 数据结构与内存模型 (Data & Memory Specs)

### 3.1 VFS 二进制格式 (Tagged Site Binary Format)
参考 v2ray 规则索引设计的自定义流式格式：
- **Header (64 bytes)**: 
    - `Magic [4]byte` ("SITE") | `Version uint8` | `EntryCount uint32` | `Checksum [32]byte` | `DictOffset uint64`。
- **Dictionary Block**: 存储所有去重后的 Tags 字符串。格式：`[Count uint32] + [ [Len uint16][String] ... ]`。
- **Payload (Streaming Entries)**:
    - `[RuleType uint8]` (0:Plain, 1:Regex, 2:Domain, 3:Full)
    - `[ValueLen uint16]` + `[Value String]`
    - `[TagCount uint8]` + `[TagIndices []uint32]`

### 3.2 内存匹配引擎 (Domain Matching Engine)
- **Domain Trie (域名后缀树)**: 
    - 核心结构。将域名按段拆分并倒序插入（如 `com.google.www`）。
    - 每个节点携带 `Tags` 载体。匹配时从根向下遍历，记录路径上所有的 `domain` 命中和终点的 `full` 命中。
- **Aho-Corasick 自动机**: 处理所有 `keyword` 规则，实现一次性 O(N) 扫描。
- **Regex Pool**: 正则表达式分组池，仅在 Trie 和 AC 未命中时作为保底手段执行。

---

## 4. 处理器规格 (Processor Specifications)

### 4.1 处理器：`site/pool/import`
- **逻辑**: 解析外部列表，执行规整化。
- **格式支持**: 
    - `geosite`: 解析标准 `geosite.dat`（需处理多 Tag 拆分）。
    - `text`: 每行一个规则，支持前缀识别（如 `full:google.com`）。
- **并发与锁调度**: **全局任务串行**，防止百万级域名解析导致内存雪崩。

---

## 5. 任务清单 (Action Plan)

### 第一阶段：基础设施 (Infrastructure & Storage)
- [ ] **Task 1: [Models]** 定义 `SiteGroup` 与 `SiteExport` 模型。实现 `render.Binder` 验证。
- [ ] **Task 2: [Tagged-Codec]** 实现支持 4 种规则类型的二进制流式编解码器。
- [ ] **Task 3: [Matching-Engine]** 开发基于 **域名后缀 Trie** 和 **AC 自动机** 的复合匹配引擎。

### 第二阶段：核心业务逻辑 (Core Services & Repositories)
- [ ] **Task 4: [Repositories]** 实现 `network/site` 命名空间下的 DB 持久化。
- [ ] **Task 5: [Services]** 实现游标预览、级联删除、审计日志接入。
- [ ] **Task 6: [Analysis-Engine]** 实装沙盒命中推演 (Hit Test) API。
- [ ] **Task 7: [Action-Handlers]** 实现 `site/pool/import` 处理器。

### 第三阶段：动态导出与逻辑合并 (Dynamic Export)
- [ ] **Task 8: [Deduplication]** 实现基于 Trie 树的子域消除算法（子域冗余过滤）。
- [ ] **Task 9: [Formatters]** 实现 `geosite.dat`, `text`, `adguard`, `dnsmasq` 渲染器。
- [ ] **Task 10: [Export-Cache]** 实装异步任务管理器与 Checksum 缓存校验。

### 第四阶段：前端与交付 (Frontend Adapters)
- [ ] **Task 11: [UI-Pools]** 开发域名池管理界面（全屏数据管理，支持搜索与无限滚动）。
- [ ] **Task 12: [UI-Analysis]** 开发域名“研判实验室”面板。
- [ ] **Task 13: [UI-Export]** 开发异步导出配置页。
