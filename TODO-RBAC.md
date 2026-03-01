# Homelab RBAC 机制补充设计与演进方案

## 1. 现状回顾 (Current State) [已更新]
基于最近的重构，RBAC 系统已演进为具备以下特性的高性能安全框架：
- **核心模型**：采用“单管理员 + 多 ServiceAccount”模式。`root` 拥有全局最高权限，`ServiceAccount` (SA) 受 RBAC 规则严格限制。
- **高性能缓存 (LRU)**：集成了 `golang-lru/v2`，对 `Role` 和 `Token` 实现了强类型缓存；对 `RoleBinding` 实现了读写锁保护的切片缓存。
- **细粒度资源控制 (Fine-Grained)**：采用资源/实例合并语法（如 `dns/example.com`），支持通配符 `*`。
- **解耦鉴权**：`RequirePermission` 中间件仅负责准入，业务 Handler 通过 Context 获取权限工具类执行最终精确匹配。
- **审计系统**：建立了基于手动上报模式的审计日志系统，支持全屏搜索。

## 2. 已完成的任务 (Completed Tasks)
- [x] **Task 1: 高性能缓存层** - 引入 LRU 库并重构 `rbac.go` 的存储读取逻辑。
- [x] **Task 2: 细粒度资源语法** - 修改 `PolicyRule` 支持 `resource/instance` 格式，重写匹配算法。
- [x] **Task 3: 鉴权机制解耦** - 中间件通过 Context 传递权限工具类 `ResourcePermissions`。
- [x] **Task 4: 审计日志系统** - 实现 `AuditLogger` 注入及业务模块手动上报，完成前端“审计日志”页面。
- [x] **Task 5: Token 活跃度追踪** - 在 `AuthMiddleware` 中实现节流异步更新 `LastUsedAt`。
- [x] **Task 6: 代码规范化** - 创建泛型 `SyncMap` 工具类，移除所有手动排序逻辑。

## 3. 待处理的问题与后续演进 (Remaining & Evolution)

### 3.1 权限配置 UI 增强
- **目标**：在 `CreateRoleDialog` 中，当输入 `dns/` 时，根据已有域名列表提供智能补全建议，减少手动输入成本。

### 3.2 权限评估工具 (Policy Simulator)
- **功能描述**：在前端提供一个调试工具，允许管理员输入一个 SA 名和目标资源路径，实时预览最终的权限评估结果（哪些资源被 AllowedAll，哪些受限于实例），方便排查配置逻辑。

### 3.3 Token 安全增强 (Long-term)
- **有效期管理**：为 SA Token 增加可选的过期时间 (`ExpiresAt`)。
- **快速封禁**：在不删除 SA 的情况下，通过 `enabled` 状态快速禁用某个 SA 的所有 Token。

## 4. 修正的任务拆分 (Revised Task Breakdown)

- [ ] **Task 7: 角色配置自动补全** - 结合后续业务模块实现实例名称的动态感知。
- [ ] **Task 8: 权限模拟器 API** - 实现一个模拟评估接口，返回指定 SA 的有效权限详情。
- [ ] **Task 9: 前端模拟器页面** - 可视化展示权限匹配链路。

---
*注：本项目坚持单用户、多 SA 的极简架构，优先保证核心业务逻辑（如 DNS）的开发进度。*
