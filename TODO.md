# Homelab 重构 TODO

## 目标

- 按能力边界拆分模块，而不是按“大域聚合”做超级模块。
- 路由注册独立成 `pkg/routes/*`，模块负责生命周期，controller 只负责 handler。
- 保留共享实现下沉，但不牺牲模块可维护性。
- 不考虑兼容性，按全新项目直接重写。

## 目标结构

### 模块

```text
pkg/modules/
  core/discovery
  core/session
  core/rbac
  core/auth
  core/audit

  network/dns
  network/ip
  network/site
  network/intelligence

  workflow
```

### 路由

```text
pkg/routes/
  core.go
  network.go
  workflow.go

  core/
    discovery.go
    session.go
    rbac.go
    auth.go
    audit.go

  network/
    dns.go
    ip.go
    site.go
    intelligence.go
```

## 模块职责

### Core

- `core/discovery`
  - 暴露 discovery API
  - 消费 registry

- `core/session`
  - 会话查询
  - 会话吊销
  - 登出

- `core/rbac`
  - role
  - rolebinding
  - serviceaccount
  - simulate
  - resource/verb suggest

- `core/auth`
  - 登录
  - 当前身份信息
  - auth 路由入口

- `core/audit`
  - 审计查询
  - 审计清理

### Network

- `network/dns`
  - domain
  - record
  - export
  - soa

- `network/ip`
  - pool
  - export
  - sync
  - analysis

- `network/site`
  - pool
  - export
  - sync
  - analysis

- `network/intelligence`
  - source
  - sync
  - task

### Workflow

- `workflow`
  - workflow
  - instance
  - log
  - webhook
  - trigger

## 设计约束

- 允许 `routes/core.go`、`routes/network.go` 做聚合注册。
- 不允许再引入 `modules/core`、`modules/network` 这种吞并子能力生命周期的大模块。
- 每个模块只管理自己的 discovery 注册。
- 每个模块只启动自己的 runner、cron、task manager。
- controller 不再承担模块装配职责。
- service 共享逻辑可以下沉到公共包，但模块边界不跟着消失。

## 模块接口

```go
type Module interface {
    Name() string
    RegisterRoutes(r chi.Router)
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
}
```

建议模块名使用能力全名：

- `core.auth`
- `core.rbac`
- `core.audit`
- `network.dns`
- `network.ip`
- `network.site`
- `network.intelligence`
- `workflow`

## 重构阶段

### Phase A：恢复能力切片模块

- 删除当前粗粒度的 `pkg/modules/core`
- 删除当前粗粒度的 `pkg/modules/network`
- 重建：
  - `pkg/modules/core/discovery`
  - `pkg/modules/core/session`
  - `pkg/modules/core/rbac`
  - `pkg/modules/core/auth`
  - `pkg/modules/core/audit`
  - `pkg/modules/network/dns`
  - `pkg/modules/network/ip`
  - `pkg/modules/network/site`
  - `pkg/modules/network/intelligence`
  - `pkg/modules/workflow`

验收：

- `bootstrap` 只装配能力切片模块
- 不再存在新的超级 `core/network` 模块

### Phase B：引入独立路由层

- 新建 `pkg/routes/*`
- 从模块中移出 HTTP 注册细节
- 模块只调用对应的 route registrar
- 保留：
  - `pkg/routes/core.go`
  - `pkg/routes/network.go`
  - `pkg/routes/workflow.go`

验收：

- `pkg/modules/*` 不直接堆叠大量 `chi` 路由细节
- `pkg/controllers/*` 不再暴露路由装配职责

### Phase C：对齐 discovery 与后台任务

- discovery 注册代码迁到对应模块边界
- runner/cron/task manager 迁到对应模块边界
- 删除跨能力代管逻辑

验收：

- `network/ip` 只启动 IP 的同步任务
- `network/site` 只启动 Site 的同步任务
- `network/intelligence` 只启动 Intelligence 的同步任务
- `workflow` 只启动 workflow 的 trigger/self-healing

### Phase D：清理 service 边界

- 继续保留 `services/rules` 作为共享实现层
- 必要时新增 `services/core`、`services/network` 公共层
- 删除仅用于旧边界的 service 入口壳
- 不再新增 `dns/ip/site/intelligence` 的重复 discovery/bootstrapping 入口

验收：

- 共享实现下沉
- 模块边界不被共享实现反向吞掉

## 当前重构原则

- 不保留旧接口兼容
- 不保留旧路由兼容
- 不保留旧模块聚合方案
- 优先删除错误边界，再补正确边界

## 禁止事项

- 禁止重新引入 `pkg/modules/core/module.go` 这种承载全部 core 生命周期的超级模块
- 禁止重新引入 `pkg/modules/network/module.go` 这种承载全部 network 生命周期的超级模块
- 禁止把 discovery 继续散落在无关 service 入口文件中
- 禁止 controller 再承担 route grouping 和模块注入职责
