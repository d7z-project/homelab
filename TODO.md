/# Network Site 增强功能实现计划 (Implementation Plan for Network Site Enhancements)

为了实现与 `network/ip` 模块的功能对等，并满足对 `geosite.dat` 的导入导出需求，建议按以下结构进行开发。

## [x] 1. 核心协议支持 (Protobuf / V2Ray)
- **目标**: 在不引入重量级依赖的情况下，支持 `geosite.dat` 的二进制编解码。
- **文件**: `backend/pkg/services/site/v2ray_pb.go`
- **内容**:
    - 使用 `google.golang.org/protobuf/encoding/protowire` 手动处理 Protobuf 字段。
    - 实现 `BuildV2RayGeoSite(w io.Writer, sites map[string][]models.SitePoolEntry)`。
    - 实现 `ParseV2RayGeoSite(data []byte, targetCategory string, importAll bool) ([]parsedGeoSiteEntry, error)`。
    - 映射规则类型：Plain(0)->Keyword, Regex(1)->Regex, Domain(2)->Domain, Full(3)->Full。

## [x] 2. 增强导入功能 (Import Processor)
- **目标**: 支持从本地或上传的 `geosite.dat` 文件导入域名规则。
- **文件**: `backend/pkg/services/site/processor.go`
- **变更**:
    - 在 `ImportProcessor.Execute` 中实装 `format == "geosite"` 的分支。
    - 支持从参数中获取 `category`（即 v2ray 中的 country_code），若未指定则导入全部并以 category 作为 Tag。
    - 保持现有的 Punycode 转换与清洗逻辑。

## [x] 3. 增强导出功能 (Dynamic Export)
- **目标**: 支持将域名池内容导出为标准的 `geosite.dat` 格式。
- **文件**: `backend/pkg/services/site/export.go`
- **变更**:
    - 在 `runExport` 任务中增加 `v2ray-dat` 格式支持。
    - 逻辑：按 Tag 对匹配的域名进行分组，每个 Tag 映射为一个 `GeoSite` 类别。
    - 调用 `BuildV2RayGeoSite` 生成二进制流。

## [x] 4. 引入同步策略 (Sync Policy) - 可选增强
- **目标**: 支持定时从远程 URL（如 GitHub 上的社区 geosite 列表）同步数据。
- **文件**:
    - `backend/pkg/models/site.go`: 定义 `SiteSyncPolicy` 结构体（包含 SourceURL, Cron, Mode, Format 等）。
    - `backend/pkg/repositories/site/repo.go`: 实现同步策略的 CRUD。
    - `backend/pkg/services/site/service_sync.go`: 实现 `SyncManager`，负责 Cron 调度与异步下载解析。
- **支持格式**: `text` (每行一个), `geosite` (v2ray dat)。

## [x] 5. 前端适配 (Frontend)
- **目标**: 在 UI 上提供导出选项与同步配置界面。
- **变更**:
    - `frontend/src/app/pages/site`: 在导出对话框中增加 `V2Ray Dat (geosite.dat)` 选项。
    - (若实现同步) 增加“同步策略”管理页面。

---
**审核要点**:
1. `v2ray_pb.go` 是否需要单独提取到 `pkg/common` 以供 `ip` 和 `site` 共用部分逻辑？（建议保留在各自目录，因为 GeoIP 和 GeoSite 的 Protobuf 结构虽相似但定义不同）。
2. 子域消除逻辑 (`Subdomain Deduplication`) 是否应在 `geosite` 导出时强制开启？（建议开启，以减小文件体积）。
