# Gemini 项目上下文

## 已实现功能与架构决策

### 1. 认证与会话管理 (Auth & Session)
- **核心逻辑**：位于 `backend/pkg/auth/auth.go`。使用 `common.DB.Child("auth", "sessions")` 存储 Session。
- **Context 规范化**：定义了 `AuthContext` 结构体，通过 `auth.AuthContextKey` 注入。使用 `auth.FromContext(ctx)` 获取身份信息（`root` 或 `sa`）。
- **注销机制**：支持服务端 Session 销毁。前端 `authInterceptor` 全局拦截 401 错误并触发重定向。

### 2. RBAC 权限系统 (Kubernetes Style)
- **模型定义**：`ServiceAccount`, `Role`, `RoleBinding` 位于 `backend/pkg/auth/rbac.go`。
- **存储拓扑**：统一使用 `Child` 抽象层。
    - `auth/serviceaccounts`: SA 元数据。
    - `auth/tokens`: `token -> sa_name` 的快速索引。
    - `auth/roles`: 权限规则（Resource + Verbs，支持 `*`）。
    - `auth/rolebindings`: 关联 SA 与 Role。
- **鉴权逻辑**：`root` 用户在中间件层直接放行，SA 经过 `RequirePermission` 中间件进行规则匹配。

### 3. 前端 UI 规范
- **视觉风格**：基于 Angular Material Design。
### 4. 开发工作流
- **API 同步**：修改后端 Swagger 注解后，需运行 `go generate ./...` 更新文档，随后在 `frontend` 运行 `npm run generate-api` 同步客户端。
- **路由保护**：新资源应在 `backend/route.go` 中通过 `routers.AuthMiddleware` 保护，并视情况嵌套 `routers.RequirePermission`。

## 待办与注意事项
- 目前存储驱动在内存模式下对 `Child` 的模拟可能导致单元测试隔离性问题，生产环境（Redis/Etcd）表现正常。
- 权限管理页面目前支持基础的 CRUD，更复杂的规则编辑器（如多规则配置）暂未实现。
