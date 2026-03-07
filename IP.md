# IP 池与动态导出过滤引擎 (IP Pool & Dynamic Export Engine)

## 1. 业务目标 (Business Goal)
构建一个支持多标签 (Tag)、极速检索的 IP 池管理系统，并提供基于表达式 (`go-expr`) 的无状态动态导出能力。**核心原则：DB 存元数据，VFS 存池数据，缓存驱动导出，基数树支撑查询。**

---

## 2. 核心架构设计 (Core Architecture Design)

严格遵循 `GEMINI.md` 规定的四层分层架构与基础设施规范。

### 2.1 IP 池管理 (IP Pool Management)
- **元数据 (DB)**: 存储 `IPGroup` 的基本信息（UUID、名称、描述、数据指纹 Checksum 等）。
    - **隔离规范**: Repository 层使用 `common.DB.Child("network", "ip", "groups")` 进行隔离。
- **实体数据 (VFS)**: 地址与 CIDR 列表存储于 `vfs://network/ip/pools/{uuid}.bin`（基于 `common.FS`）。
    - **安全删除**: 删除 `IPGroup` 前，校验是否有 `IPExport` 或 `IPSyncPolicy` 引用。删除时级联删除对应的 VFS `.bin` 文件。
- **多维度标签 (Tags)**: 记录附加 `Tag`。支持导入相同 IP/CIDR 并通过 Tag 区分。
- **数据预览 (Preview)**: 采用 **游标分页 (Cursor-based)**，服务端返回 Byte Offset 游标，实现 O(1) 性能的流式数据展示。

### 2.2 动态过滤与多格式导出 (Dynamic Export)
- **多格式支持**: 
    - `application/vnd.v2ray.geoip`: `geoip.dat` 格式（由 `v2ray_pb.go` 构建）。
    - `text/plain`: 换行符分隔的 CIDR 列表。
    - `application/json`: 结构化数据。
    - `application/yaml`: YAML 列表格式。
- **异步任务驱动**: 导出任务由 `ExportManager` 管理。
    - **进度与缓存复用**: 任务 Checksum 由 `Hash(Rule + GroupChecksums + Format)` 决定。命中缓存且物理文件存在时，直接返回已有 TaskID。
    - **抢占机制 (Cancel & Replace)**: 触发新生成任务时，自动取消同配置下仍在运行的旧任务。
- **安全性**: 导出结果下载受 `GetPermission` 保护。

### 2.3 辅助研判能力 (Analysis Utilities)
- **内存基数树 (Radix Tree)**: 维护高性能前缀树，叶子节点存储 Tag 集合。
- **命中推演 (Hit Test)**: 提供 API，输入 IP/CIDR，快速判断其命中的网段及其 Tags。
- **情报查询 (ASN/Org)**: 集成 MaxMind 基础库 (`.mmdb`)，提供归属地与 ASN 查询。

### 2.4 数据同步 (Data Sync)
- **同步策略**: 支持基于 Cron 的定时同步。
- **SSRF 防护**: 所有拉取行为必须经过 `validateSourceURL` 校验，禁止探测内网敏感地址。

---

## 3. 数据结构规格 (Specifications)

### 3.1 VFS 二进制格式 (Tagged Binary Format)
- **Header (紧凑布局)**: `Magic (4B)` | `Version (1B)` | `EntryCount (4B)` | `Checksum (32B)` | `DictOffset (8B)`。
- **Dictionary**: `[TagCount (4B)]` + `[Len (2B) + Bytes] * N`。
- **Payload**: `[Family (1B)]` + `[IP (4/16B)]` + `[Mask (1B)]` + `[TagCount (2B)]` + `[TagIndices (4B * N)]`。

### 3.2 导出缓存机制
- **Checksum 计算**: `SHA256(Rule + Format + Sorted(GroupIDs + GroupChecksums))`。
- **存储位置**: `common.TempDir` 下的 `temp/export_{taskID}.{format}`。

---

## 4. 模块合规性 (Compliance)

1. **权限检查 (Instance-Level)**:
   - `network/ip/{id}` 权限控制 IP 池管理。
   - `network/ip/export/{id}` 权限控制特定导出配置。
   - 所有关键操作记录 `commonaudit.Log`。
2. **错误处理**: 统一通过 `controllers.HandleError` 分发，权限拒绝必须包装 `commonauth.ErrPermissionDenied`。

---

## 5. API 路由规范 (API Endpoint Paths)

- **IP 池管理 (IP Pools)**: `/api/v1/network/ip/pools`
- **命中推演与查询 (Analysis)**: `/api/v1/network/ip/analysis`
- **动态导出 (Exports)**: `/api/v1/network/ip/exports`
- **下载导出**: `/api/v1/network/ip/exports/download/{taskId}`
- **数据同步**: `/api/v1/network/ip/sync`
