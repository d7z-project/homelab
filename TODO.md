# 资源 Meta 与 Status 逻辑分离设计方案 (Generation 驱动 + 冲突域分离)

## 1. 背景与动机

为了实现配置与状态的解耦，并提供类 Kubernetes 的强一致性与乐观并发控制（OCC），本方案采用双计数器架构。

- **架构清晰**：通过泛型容器物理隔离配置（Meta）与状态（Status）。
- **冲突域分离**：用户的配置修改（Meta）与系统的状态更新（Status）不应互相阻塞。
- **单调版本指纹**：放弃不可靠的时间戳，采用单调递增的整数 `Generation` 作为版本指纹。
- **平滑路径**：移除路径中的中置版本号，采用扁平路径以优化前缀扫描性能。

## 2. 存储拓扑与模型设计

### 2.1 存储路径

- **路径规则**: `{module}/{Model}/v1/{id}` (例如：`network/IPPool/v1/example1`)

### 2.2 资源规范与容器结构 (`Resource[M, S]`)

资源配置（Meta）与状态（Status）的结构体严格按 `{Model}V1Meta` 与 `{Model}V1Status` 命名（例如 `IPPoolV1Meta`, `IPPoolV1Status`），由泛型容器包裹：

```go
type Resource[M any, S any] struct {
    ID              string `json:"id"`
    Meta            M      `json:"meta"`
    Status          S      `json:"status"`
    Generation      int64  `json:"generation"`      // 配置版本：仅在 Meta 变更时递增
    ResourceVersion int64  `json:"resourceVersion"` // 对象总版本：任何变更（Meta/Status）均递增
}
```

## 3. 逻辑架构设计

### 3.1 冲突检测逻辑 (Conflict Resolution)

- **Meta 更新 (PatchMeta)**: 强制校验 `Generation`。若请求携带的 `Generation > 0` 且不一致，返回 `409 Conflict`。更新后 `Generation` 与 `ResourceVersion` 均递增。
- **Status 更新 (UpdateStatus)**: 非阻塞写入，不校验 `Generation`。仅递增 `ResourceVersion`。

### 3.2 多级校验体系 (Validation Strategy)

为了确保数据一致性并解耦业务逻辑，采用以下三层校验：

| 维度                      | 校验主体                 | 职责范围                             | 触发时机     |
| :------------------------ | :----------------------- | :----------------------------------- | :----------- |
| **1. Schema (格式)**      | `DTO` (render.Binder)    | 必填项、正则、枚举范围、字段预处理。 | API 参数绑定 |
| **2. Integrity (自洽性)** | `Meta` (ConfigValidator) | 内部逻辑矛盾校验（不依赖外部 DB）。  | 仓库层写入前 |
| **3. Business (业务)**    | `Service` 层             | 全局冲突检查、权限校验、引用存在性。 | 调用仓库前   |

#### 3.2.1 仓库层“最后的防线”

`BaseRepository` 在执行 `Save` 或 `PatchMeta` 时，必须通过类型断言自动调用 `Meta.Validate(ctx)`。

- 若校验失败，必须回滚 `db.Cow` 事务并返回 `400 Bad Request` 相关的包装错误。

## 4. 核心接口定义

```go
// ConfigValidator 定义了 Meta 模型必须实现的自洽性校验
type ConfigValidator interface {
    // Validate 执行不依赖外部环境的内部逻辑校验
    Validate(ctx context.Context) error
}
```

## 5. 实施路线图

### 第一阶段：基础设施实现

- [x] 在 `pkg/models/` 定义 `Resource[M, S]` 容器及 `ConfigValidator` 接口。
- [x] 在 `pkg/common/` 实现类似 k8s client-go 的 `BaseRepository[M, S]`：
  - [x] 强制 `NewBaseRepository(module, model)` 拼装类似于 K8s API 的 `{module}/{model}/v1/{id}` 存储路径。
  - [x] 在 `db.Cow` 闭包内集成 `Generation` 冲突检测。
  - [x] 自动触发 `ConfigValidator.Validate()`。
  - [x] 抽象封装内置游标流的 `List` 分页获取方法。
- [x] 封装标准的错误类型：`ErrConflict` (409) 和 `ErrInvalidConfig` (400)。

### 第二阶段：业务模型重构 (全新系统，不考虑迁移)

- [x] **IP 模块**: 重构 `IPPool` 结构，实现 `Validate` 接口校验 CIDR/Gateway 匹配逻辑。
- [x] **DNS 模块**: 重构 `Domain` 和 `Record` 结构，实现 `Validate` 接口校验域名格式。
- [x] **Intelligence 模块**: 重构 `IntelligenceSource` 结构，实现 `Validate` 接口。
- [x] **RBAC 模块**: 重构 `Role`, `ServiceAccount`, `RoleBinding` 结构，实现 `Validate` 接口。
- [ ] **Site 模块**: 重构 `SiteGroup`, `SiteSyncPolicy` 等资源。
- [ ] **Actions 模块**: 重构 `Workflow` 结构。

### 第三阶段：Service 层适配

- [x] 调整 `Update` 逻辑，要求调用方从 DTO 获取并传递 `Generation` (在主要模块中已完成)。
- [x] 确保业务逻辑校验（如 IP 冲突）在调用 `PatchMeta` 之前完成。
- [x] 统一仓库层 `Save`/`PatchMeta` 的校验逻辑。

## 6. 优势总结

- **强一致性**：`Generation` 确保并发编辑时不会覆盖他人工作。
- **数据合法性**：三层校验确保非法配置无法落地。
- **高性能写**：冲突域分离避免了高频状态更新阻塞用户配置修改。
