# IP 池与动态导出过滤引擎 (IP Pool & Dynamic Export Engine)

## 1. 业务目标 (Business Goal)
构建一个支持多标签 (Tag)、极速检索的 IP 池管理系统，并提供基于表达式 (`go-expr`) 的无状态动态导出能力。**核心原则：DB 存元数据，VFS 存池数据，缓存驱动导出，基数树支撑查询。**

---

## 2. 核心架构设计 (Core Architecture Design)

严格遵循 `GEMINI.md` 规定的四层分层架构与基础设施规范。

### 2.1 IP 池管理 (IP Pool Management)
- **元数据 (DB)**: 存储 `IPGroup` 的基本信息（UUID、名称、描述、数据指纹 Checksum 等）。
    - **ID 生成**: 标识 ID (UUID) 必须由后端在创建时自动生成，不允许用户自定义。
    - **隔离规范**: Repository 层必须使用 `common.DB.Child("network", "ip")` 进行命名空间隔离。
- **实体数据 (VFS)**: 实际的 IPv4/v6 地址与 CIDR 列表存储于 `vfs://network/ip/pools/{uuid}.bin`（基于 `common.FS`，默认 memory:// 并使用 `BasePathFs` 封装）。
    - **安全删除**: 删除 `IPGroup` 记录前，必须校验是否有 `IPExport` 规则依赖该池。如果存在引用则**拒绝删除**。允许删除时，必须**级联删除**对应的 VFS `.bin` 实体数据文件。
- **多维度标签 (Tags)**: 允许导入的每一条记录附加 `Tag`。不同来源导入的相同 IP/CIDR 可以共存并通过 Tag 进行区分。
- **条目级管理 (Entry Management)**: 支持在特定地址池内直接新增、修改、删除独立的 IP/CIDR 记录及其关联的 Tag，支持细粒度的资源管理。
    - **禁止重复**: 新增条目时，必须校验该 IP/CIDR 是否已存在于当前地址池中，防止重复添加。
    - **修改限制**: 已存在的条目仅允许修改其关联的 Tag，禁止修改 IP/CIDR 本身。
- **数据预览与检索 (Preview & Search)**: 采用 **游标分页 (Cursor-based)** 替代 Offset/Limit，服务端返回下一条记录的 Byte Offset，实现 O(1) 性能的流式数据展示。支持基于前缀或 Tag 的文本搜索过滤。客户端需基于此实现全屏管理窗口下的滚动加载 (Infinite Scroll)。

### 2.2 动态过滤与多格式导出 (Dynamic Export)
- **无状态持久化**: 导出配置 (`IPExport`) 仅在 DB 中存储过滤规则，**不实际在 VFS 中生成永久物理导出文件**。
- **多格式支持 (Export Formats)**: 导出 API 必须支持通过参数（或 Accept 头）指定格式。
    - `application/vnd.v2ray.geoip`: 导出为标准的 `geoip.dat` 格式，供 v2ray/Xray 等代理软件直接挂载使用。
    - `text/plain`: 换行符分隔的纯 CIDR 列表，供 iptables/nftables/ROS 等基础防火墙消费。
    - `application/json`: 结构化数据，包含附带的 Tag 信息。
    - `application/yaml`: 系统标准配置格式。
- **异步任务驱动 (Async Task Model)**: 导出操作因可能涉及百万级数据的读取和合并，必须采用后台异步任务执行。
    - **API 拆分**: 提供独立的“触发生成”与“获取下载链接”接口。
    - **进度与缓存复用**: 查询接口需返回包含浮点数进度的任务状态。若源数据和过滤规则未变，直接返回缓存（进度为 1.0）。
    - **抢占与最新保留 (Cancel & Replace)**: 与 IP 池导入强制串行不同，每次对同一个导出配置调用“触发生成”接口时，系统必须**自动 Cancel 取消其上一次仍在进行中的生成任务**，仅保留并执行最新触发的任务，以防止无意义的 CPU 资源浪费。
- **AST 预分析与降级 (Optimization)**: 针对海量数据，静态分析 `go-expr` 语法树。如果表达式仅为 `Tag in [...]`，直接降级为原生的字典匹配，绕过运行时求值开销。
- **带有 TTL 的依赖感知缓存**:
    - 结合 `common.TempDir` 下的临时目录与 LRU 策略存储生成的计算结果，并配置 TTL (如 24 小时)。
    - **缓存一致性**: 缓存的 Key 由“导出规则哈希 + 依赖的源 IP 池 Checksum + 导出格式”共同决定。
    - **垃圾回收 (GC)**: 预留 `TriggerGC()` 接口。具体基于 TTL 和废弃缓存文件的后台清理调度逻辑作为后续独立模块设计 (TODO)。

### 2.3 辅助研判能力 (Analysis Utilities)
- **并发安全的全局缓存**: 命中推演不能每次“临时构建树”。系统维护一个基于使用频率淘汰的 Radix Tree 全局缓存池。对树的读写更新必须通过 `atomic.Value` 或 `sync.RWMutex` 进行严格并发控制。
- **命中推演 (Hit Test)**: 提供沙盒 API，输入单个 IP 或 CIDR，快速判断其是否命中特定 IP 池，并返回匹配的具体网段及其附带的 Tags。
- **情报查询 (ASN/Org)**: 提供单 IP 归属查询接口，集成 MaxMind 基础库，返回 ASN (自治系统编号)、组织名称及基础地理信息。
- **基础库管理 (MMDB lifecycle)**: 将 MaxMind `.mmdb` 文件的更新同样纳入自动引擎范畴，通过特定的 `ip/download/mmdb` 动作处理器接管。

### 2.4 全局审计与监控 (Audit & Observability)
- **全局审计日志**: 所有的关键行为必须接入 `commonaudit.Log`，包括但不限于：创建/修改 `IPGroup` 规则、触发导出缓存生成、手动进行命中推演等。

---

## 3. 数据结构与内存模型 (Data & Memory Specs)

### 3.1 VFS 二进制格式 (Tagged Binary Format)
针对带 Tag 的设计，优化二进制存储，采用“常量字符串字典”避免冗余：
- **Header**: `Magic` | `Version` | `EntryCount` | `Checksum` | `DictOffset`。
- **Dictionary Block (常量字符串字典)**: 存储所有去重后的 Tag 字符串。格式为 `[TagCount] + [Tag1_Len, Tag1_Bytes] + [Tag2_Len, Tag2_Bytes] + ...`。
- **Payload**: `[CIDR 数据块 (IP+Mask)]` + `[Tags 索引数组 (指向 Dictionary Block 的 Index)]`。
- **Index Block (Optional)**: `[Tag 倒排索引块]` (记录特定 Tag Index 对应的 IP Payload 偏移量，加速过滤)。

### 3.2 内存基数树 (Tagged Radix Tree)
由于 `go4.org/netipx` 库中的 `IPSet` 仅支持单纯的 IP 集合判断，无法在节点中存储额外的值 (Value/Tags)，因此系统在内存索引层面采取**双轨设计**：
- **研判与查询树 (Analysis Trie)**:
    - 内存中维护高性能的 Radix Tree，其叶子节点必须能够存储该网段关联的 **Tag 集合**。
    - **实现方案**: 必须基于 Go 1.18 的 `netip.Prefix` 手写实现，或引入支持关联泛型 Payload 的开源前缀树库 (如基于 `Trie` 的路由库)。必须暴露 `Insert(prefix netip.Prefix, tags []string)` 和 `Lookup(ip netip.Addr) []string` 接口。
- **导出合并引擎 (Export IPSet)**:
    - 原生 `netipx.IPSet` 及其 `Builder` 仅专门用于不携带 Tag 诉求的“导出引擎” (`Dynamic Export`) 环节，专门负责海量散乱 IP 的交并差高阶运算与最终 CIDR 的极致压缩。

---

## 4. 处理器规格 (Processor Specifications)

集成至通用自动化引擎 (Actions Engine)：
*(注：任务执行期间必须通过 `commonauth.WithAuth` 注入工作流指定的 `ServiceAccountID`)*

### 4.1 处理器：`ip/pool/import`
- **逻辑**: 将获取到的外部数据执行 Tag 规整化后，与现有数据进行全量合并运算 (Read-Merge-Write)，生成全新的带 Tag 二进制格式快照，并执行 `Temp-and-Rename` 原子覆盖至 `vfs://network/ip/pools/{group_uuid}.bin`。
- **并发与锁调度**:
    - **全局任务串行 (Strict Serialization)**: 所有涉及 IP 池数据更新的任务（不论针对相同还是不同的 `group_uuid`），都必须统一放入全局队列，实施**严格的串行排队处理**。这旨在防止并发全量重写导致系统 OOM 内存毛刺，并消除同池并发修改引发的数据状态冲突。
- **输入参数 (Action Inputs)**:
    - `filePath`: 待导入的本地逻辑文件路径（如上游步骤生成的临时文件）。
    - `format`: 输入文件的格式。必须支持：
        - `geoip`: 解析标准的 `geoip.dat` (v2ray 格式) 或 `mmdb` 格式。
        - `text`: 解析纯文本换行符分隔的 IP/CIDR 列表。
        - `json`/`csv`: 解析通用开源情报库（如黑客/C2 节点数据库）的特定结构。
    - `mode`: 导入模式。`append` (附加) 或 `delete` (删除)。
    - `targetPool`: 目标的 IP 池 UUID (`group_uuid`)。
    - `defaultTags`: 数组。导入时为所有数据附加的默认 Tag；在 `delete` 模式下，如果指定了该参数，则仅从池中删除带有这些 Tag 的记录（按 Tag 精准消除），若未指定则删除匹配的 IP 条目本身。
- **更新信号**: 写入完成后，更新 DB 中该 Group 的最后更新时间与 Checksum，自动使得依赖该池的**导出缓存失效**，并触发拦截引擎热加载。

### 4.2 处理器：`ip/download/mmdb`
- **逻辑**: 专用于下载与校验更新 MaxMind ASN/GeoIP 数据库文件，并执行原子替换。提供给情报查询引擎使用。
- **参数注入**: 无需特殊的全局鉴权配置。MaxMind 所需的 `AccountID` 和 `LicenseKey` 等凭证直接作为该 Action 步骤的标准 `params` 传入，保持自动化引擎的通用性。

---

## 5. 模块组成与架构合规性 (System Modules & Compliance)

遵循 `GEMINI.md` 的强制性架构规范：

1. **模型层 (Models)**
   - `IPGroup`, `IPExport` 等实体字段必须带有 camelCase 标签，且**必须实现 `render.Binder`** 进行基础格式与非空校验。
2. **业务服务层 (Services / IP-Pool-Service / Analysis-Engine)**
   - 必须通过 `commonaudit.Log` 记录所有影响配置与数据流向的关键操作。
   - **错误处理**: 涉及权限拦截的地方，必须使用 `fmt.Errorf("%w: ...", commonauth.ErrPermissionDenied)` 包装错误。
3. **控制器层 (Controllers)**
   - **路由定义**: 强制使用 `r.With(middlewares.RequirePermission(...)).Method(path, handler)` 链式注入权限。严禁包反向依赖。
   - **统一响应**: 所有 Handler 通过 `controllers.HandleError(w, r, err)` 进行错误探测与 401/403/500 的分发。

### 5.1 API 路由规范 (API Endpoint Paths)
为了防止多人协作开发时出现 URL 风格不一致的问题，本模块必须遵循以下统一定位的 RESTful 资源路径规范：
- **IP 池管理 (IP Pools)**: `/api/v1/network/ip/pools`
- **命中推演与查询 (Analysis)**: `/api/v1/network/ip/analysis`
- **动态导出 (Exports)**: `/api/v1/network/ip/exports`

---

## 6. 任务清单 (Action Plan)

严格遵循 `GEMINI.md` 核心开发工作流。

### 第一阶段：基础设施 (Infrastructure & Storage)
- [ ] **Task 1: [Models]** 定义 `IPGroup` 与 `IPExport` 模型。实现 `render.Binder` 验证。
- [ ] **Task 2: [Tagged-Codec]** 实现支持 Tag 存储的二进制编解码器，**必须支持流式读取 (`io.Reader`) 接口**。
- [ ] **Task 3: [MMDB-Manager]** 实现 MaxMind 基础库的 VFS 托管 (`common.FS`) 与加载逻辑。

### 第二阶段：核心业务逻辑 (Core Services & Repositories)
- [ ] **Task 4: [Repositories]** 实现基于 `common.DB.Child("network", "ip")` 的 DB 持久化。
- [ ] **Task 5: [Services]** 实现基于游标的数据预览接口。增加单池容量上限检查，并深度集成 `commonaudit.Log`。向 Discovery 服务注册 `IPGroup` 与 `IPExport` 的 LookupFunc（注意防止除零 Panic）。
- [ ] **Task 6: [Analysis-Engine]** 实现使用 `atomic.Value` 保护的全局 LRU 缓存，开发命中推演与 ASN 归属查询 API。
- [ ] **Task 7: [Action-Handlers]** 实现 `ip/pool/import` (含 Tag 规整化) 及 `ip/download/mmdb` 处理器。
- [ ] **Task 8: [Controllers]** 挂载链式路由，统一接管 `HandleError`。

### 第三阶段：动态导出与缓存 (Dynamic Export & Testing)
- [ ] **Task 9: [Expr-Engine]** 集成 `go-expr`，实现 AST 降级加速，并支持 `text/plain`, `json`, `yaml` **多格式渲染输出**。
- [ ] **Task 10: [Export-Cache]** 实现本地临时文件 (`common.TempDir`) 的多格式导出缓存系统。
- [ ] **Task 11: [Tests]** 编写涉及大数据量加载的防内存毛刺基准测试。在 `security_instance_test.go` 中验证单实例 Deny 隔离。
- [ ] **Task 12: [Sync]** 运行 `make backend-gen` 同步 API 定义。

### 第四阶段：前端与交付 (Frontend Adapters)
- [ ] **Task 13: [UI-Pools]** 遵循 M3 规范开发 IP 池管理界面。遇到 403 提供明确的无权限提示。
- [ ] **Task 14: [UI-Analysis]** 开发“IP 研判实验室”面板。
- [ ] **Task 15: [UI-Export]** 开发导出配置页，提供直观的导出选项。触发后在后台异步执行导出任务，完成后在界面提供文件下载链接。
- [ ] **Task 16: [Build]** 运行 `make all` 全栈编译验证。
