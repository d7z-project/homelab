# Homelab DNS 管理功能开发任务拆分

## 阶段 1：后端核心逻辑与数据模型 (Backend Core)

本阶段的目标是建立 DNS 管理的基础数据结构，并利用现有的 kv 存储系统（boltdb，通过 `common.DB` 抽象）完成数据的持久化操作。本阶段不涉及对外 API 暴露。

### 详细任务清单

#### 1.1 数据模型定义 (`backend/pkg/dns/models.go`)
- **定义 Domain (域名) 结构体**:
  - `ID`: 唯一标识符（UUID）。
  - `Name`: 域名（如 `example.com`），需唯一。
  - `Status`: 状态（例如：`active`, `inactive`）。
  - `Comments`: 备注说明。
  - `CreatedAt`: 创建时间。
  - `UpdatedAt`: 更新时间。
- **定义 Record (解析记录) 结构体**:
  - `ID`: 唯一标识符（UUID）。
  - `DomainID`: 关联的域名 ID。
  - `Name`: 记录名（如 `@`, `www`, `api`）。
  - `Type`: 记录类型（枚举：`A`, `AAAA`, `CNAME`, `MX`, `TXT`, `NS`, `SRV`, `CAA` 等）。
  - `Value`: 记录值（如 `192.168.1.1` 或复杂的文本）。
  - `TTL`: 生存时间（秒）。
  - `Priority`: 优先级（仅用于 `MX` 和 `SRV` 记录，其他类型忽略）。
  - `Status`: 状态（例如：`active`, `inactive`）。
  - `Comments`: 备注说明。

#### 1.1.1 DNS 类型支持与核心策略
- **支持的常规记录类型**: 
  - `A` (IPv4)
  - `AAAA` (IPv6)
  - `CNAME` (别名)
  - `MX` (邮件交换)
  - `TXT` (文本记录)
  - `NS` (名称服务器)
  - `SRV` (服务位置)
  - `CAA` (证书颁发机构授权)
- **同名多值记录 (Round-Robin 等) 处理方式**:
  - 采用**扁平化记录 (Flat Records)** 设计。系统允许用户为同一个 `Name` 和 `Type`（如 `www` 和 `A`）创建多条独立的 `Record` 数据条目，它们拥有各自独立的 `ID`、`Value` 和状态。
  - **优势**：CRUD 操作更细粒度，用户可以单独禁用/删除某一个 IP，而不需要在一个数组字段中进行复杂的编辑。后端 DNS 服务（如 CoreDNS 插件）在查询库时，会自动将同名、同类型的多条有效记录聚合并一起返回。
- **SOA (Start of Authority) 记录处理策略**:
  - **隐式接管/自动生成**：系统在创建 `Domain` (域名) 时，**不作为常规 `Record` 开放给用户手动编辑**。如果 Homelab 作为权威 DNS 服务器对外提供服务，SOA 记录应由系统在 DNS 解析层动态计算并自动附加（包含固定的主名称服务器、管理员邮箱、动态生成的 Serial 序号及默认的 Refresh/Retry 策略）。

#### 1.4 数据一致性与验证 (Validation)
- 在保存（Save）操作前，增加基础的业务逻辑校验：
  - **格式校验**：域名格式正则校验、IP 地址格式校验（当 Type 为 A 或 AAAA 时）。
  - **CNAME 互斥校验 (RFC 1034)**：
    - 如果某个 `Name` 已经存在 `CNAME` 记录，则绝对不允许再添加同一 `Name` 下的任何其他记录。
    - 反之，如果某个 `Name` 已经存在其他记录（如 A, TXT），则不允许再为其添加 `CNAME` 记录。

---

## 阶段 2：后端 API 路由与鉴权 (Backend API)
本阶段将基于阶段一实现的核心逻辑，对外暴露符合 RESTful 风格的 HTTP API，并添加安全控制。

### 2.1 编写 DNS HTTP Handler (`backend/pkg/routers/dns.go`)
- **Domain (域名) 接口**:
  - `POST /api/v1/dns/domains`: 创建域名。
  - `GET /api/v1/dns/domains`: 获取域名列表（支持 `page`, `pageSize`, `search` 参数）。
  - `PUT /api/v1/dns/domains/{id}`: 更新指定域名（包含状态启停等）。
  - `DELETE /api/v1/dns/domains/{id}`: 删除域名（包含删除关联记录的逻辑）。
- **Record (解析记录) 接口**:
  - `POST /api/v1/dns/records`: 创建解析记录。
  - `GET /api/v1/dns/records`: 获取解析记录列表（必须支持按 `domainId` 过滤，以及 `page`, `pageSize`, `search`）。
  - `PUT /api/v1/dns/records/{id}`: 更新解析记录。
  - `DELETE /api/v1/dns/records/{id}`: 删除解析记录。

### 2.2 添加 Swagger 注解
- 为上述每个 Handler 函数添加完整的 Swaggo 注解（`@Summary`, `@Description`, `@Tags`, `@Accept`, `@Produce`, `@Param`, `@Success`, `@Router`, `@Security`）。
- 定义标准的请求/响应 DTO (Data Transfer Objects)，例如分页响应结构体 `common.PaginatedResponse`，以便前端正确解析。

### 2.3 注册路由与权限中间件 (`backend/route.go`)
- 在主路由配置中挂载 DNS 相关的端点。
- **安全加固**：
  - 对所有 DNS API 应用 `AuthMiddleware` 确保用户已登录。
  - 应用 RBAC 中间件 `RequirePermission("admin", "dns")`，确保只有具备相应权限的实体才能进行管理。

## 阶段 3：接口同步 (API Generation)
本阶段负责将后端定义的 API 规范自动转化为前端可用的强类型代码。

### 3.1 重新生成 Swagger 文档
- 执行 `go generate ./...`（通常在 `Makefile` 的 `backend-generate` 中）。这会将后端的 Swagger 注解解析并更新到 `backend/docs/swagger.json`。

### 3.2 同步前端 API 客户端
- 执行 `npm run generate-api`（基于 OpenAPI Generator）。
- 确认生成的 `dns.service.ts` 及相关的 Model 文件（如 `DnsDomain`, `DnsRecord`）已在前端项目中生成。
- 检查生成的模型属性类型是否与阶段一设计的保持一致，以便为阶段四的前端开发提供可靠的类型推断支持。

## 阶段 4：前端 UI 实现 (Frontend UI)
本阶段旨在提供与 RBAC 模块一致的 Material Design 3 高质量交互体验。

### 4.1 导航与路由集成
- **侧边栏导航**：在 `MainComponent` 的 `menuItems` 中添加“DNS 管理”父菜单。
- **子菜单支持**：配置子菜单项“域名管理”和“解析记录”，并携带对应的 `queryParams` (如 `?tab=domain` 和 `?tab=record`)。
- **路由注册**：在 `app.routes.ts` 中注册 `DnsComponent`。

### 4.2 核心页面设计 (`DnsComponent`)
- **布局风格**：复用 `.bg-surface-container-lowest` 背景、最大宽度 `max-w-7xl` 以及大圆角 (`rounded-3xl`) 的卡片式表格设计。
- **状态联动**：
  - **顶部工具栏**：根据 `UiService` 设置为随滚动隐藏 (`toolbarSticky: false`)，使 Tab 栏能在滚动时无缝吸顶。
  - **双选项卡**：使用 `mat-tab-group` 实现“域名管理”与“解析记录”的切换。
- **高级搜索交互**：
  - 复用最新的**动态浮动搜索框**：默认隐藏，通过右下角 FAB 唤出，带有全屏变暗的毛玻璃遮罩 (`backdrop-blur`)。
  - **数据信息栏**：在表格顶部显示当前数据总数、已加载数量，并在搜索时显示带清除（X）按钮的“正在搜索: xxx”胶囊标签。
  - **动态 FAB**：搜索 FAB 的图标和颜色 (`secondary` ↔ `tertiary`) 根据当前是否有搜索内容智能切换。

### 4.3 表格与数据展示
- **域名表格**：展示域名名称（等宽字体）、状态（使用 M3 颜色标签，如绿色的 active 和灰色的 inactive）、备注及操作列。
- **解析记录表格**：
  - 强依赖关联：必须先选择或在顶部过滤特定的域名，才能有效管理记录。
  - 列定义：展示 `Name`、`Type` (使用不同颜色的徽章区分 A, CNAME, TXT 等)、`Value` (超长文本需截断或提供复制按钮)、`TTL`、状态和操作。
- **滚动加载**：实现基于滚动的无限加载（Infinite Scroll），配合后端的基于分页的接口。

### 4.4 弹窗组件交互 (Dialogs)
- **`CreateDomainDialogComponent`**：
  - 用于新增或编辑域名。包含简单的名称校验和状态切换。
- **`CreateRecordDialogComponent`**：
  - **动态表单**：根据所选的 DNS `Type` 动态调整输入项。例如：选 `MX` 时显示 `Priority` 输入框；选 `A` 时对 `Value` 进行 IPv4 格式校验。
  - 提示用户**同名多值**规则（例如可以添加多个相同的 A 记录）。
- **`ConfirmDialogComponent`**：
  - 复用全局确认弹窗，用于拦截删除操作，并使用红色警告色 (`color="warn"`) 提示风险（尤其是删除域名时可能级联删除记录）。
