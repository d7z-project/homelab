# Homelab

[English](README.md)

> [!WARNING]
> 本项目为 "vibe coding" 产物，不建议用于生产环境。

一个为家庭实验室环境设计的现代、安全的基础设施管理系统。

## 功能特性

- DNS：全周期的域名与解析记录管理。
- RBAC：基于资源路径和通配符的精细化权限控制。
- 审计：完整的管理员操作行为追踪。
- 安全：支持 Root 会话与 ServiceAccount，集成 TOTP 二步验证。
- 界面：极简瑞士风格（基于 M3 与 Tailwind v4）。

## 技术栈

- 后端：Go 1.21+，Chi，BoltDB。
- 前端：Angular 17+，Material Design 3。

## 快速启动

```bash
make install
make all
```

- 后端：`cd backend && go run main.go`
- 前端：`cd frontend && npm start`

## 开发指南

在修改 API 后，请运行 `make backend-generate` 以同步 Swagger 文档及前端 API 客户端。

---
*MIT 协议 © 2026 Homelab 基础设施系统*
