# Homelab 分布式协调层 design (Distributed Coordination Design) v2.0

## 1. 背景与挑战 (Background & Challenges)
在共享 VFS (如 NFS/SMB) 的多机部署环境下，简单的分布式抽象会导致性能毛刺与数据丢失：
1. **I/O 惊群效应**: 全集群同时重载大文件（如 70MB MMDB）会导致存储带宽瞬间打满，影响在线 API 响应。
2. **事件遗漏**: 节点在网络波动、进程重启或冷启动期间，会错过 Pub/Sub 发送的实时状态变更指令。
3. **日志写回风险**: 原有的“单点胜出者最终写回”模式存在单点故障风险。若负责写回的节点在任务结束时崩溃，整场任务的日志将从 KV/内存中永久丢失。
4. **时钟偏移 (Clock Skew)**: 节点间微小的系统时钟差异会导致 Cron 任务在分布式锁释放时产生空档（跳过任务）或重叠执行。

## 2. 技术选型 (Technology Stack)
本项目基于现有的强一致性基础设施构建分布式协调抽象层：
*   **状态同步 (State)**: `kv.KV` (common.DB) - 存储集群全局版本号和元数据。
*   **互斥控制 (Lock)**: `lock.Locker` (common.Locker) - 资源争抢与分布式临界区。
*   **事件广播 (Pub/Sub)**: `subscribe.Subscriber` (common.Subscriber) - 节点间实时通知。

## 3. 核心机制设计 (Core Mechanisms)

### 3.1 混合同步模式 (Passive Sub + Active Check)
为了防止消息丢失，各节点采用双重校验机制：
*   **被动订阅 (Pub/Sub)**: 实时监听 `subscribe.Subscriber` 发出的事件（如 `RELOAD_MMDB`），实现秒级响应。
*   **主动补偿 (Active Polling/Check)**: 节点在**启动、网络连接恢复、以及执行关键业务操作前**，必须主动比对 `common.DB` 中的全局版本号。
*   **版本控制 (Versioning)**: 使用 `system:sync:version:{module}` 作为 Key，存储单调递增的 UnixNano 时间戳。

### 3.2 平滑重载与抖动 (Jittered Reload)
针对 I/O 密集型操作（如 MMDB 库加载、大规模配置文件读取）：
*   **逻辑**: 收到重载信号后，节点不在第一时间执行。
*   **实现**: 在 `[0, 1000ms]` 范围内生成一个随机延迟（Jitter）。通过时间上的错峰，平滑集群整体对共享存储的 I/O 吞吐需求。

### 3.3 租约化分布式调度 (Lease-based Scheduling)
*   **租约锁**: 使用带 TTL 的分布式锁作为执行租约。
*   **配置策略**: TTL 应设置为任务周期的 50%（例如：分钟级任务 TTL=30s）。这既能防止节点死锁导致后续任务被跳过，也能预留足够的容错空间。
*   **时钟容错**: 抢锁失败的节点会进入一个极短的随机等待期（如 50-200ms）后再次尝试，以抵消节点间 100ms 以内的时钟偏差。

## 4. 场景化解决方案 (Scenario Solutions)

### 4.1 情报库同步 (Intelligence Sync)
1.  **更新端**: `IntelligenceService` 下载成功后，更新 `common.DB` 中的版本号，并向集群广播 `RELOAD_MMDB` 事件。
2.  **消费端**: 
    *   接收事件 -> 随机延迟 0-1s -> 执行 `Reload()`。
    *   **自愈**: 如果 `Reload()` 连续失败，将本地缓存版本标记为 0，强制在下一次 API 调用时触发同步。

### 4.2 流式日志追加 (Streaming Log Append)
放弃高风险的“胜出者一次性写回”模式，改为高可靠的流式追加：
1.  **分布式暂存**: 各执行节点将日志实时 Push 到 `common.DB` 的 List 结构中（Key: `task:logs:{id}`）。
2.  **流式落盘**: 节点负责将自己产生的日志块（Chunk）异步追加到 VFS 的 `.log.tmp` 临时文件中。
3.  **最终整理**: 任务彻底结束后，由最后一个完成的节点获取分布式锁，执行一次 `VFS.Rename` 将临时日志转为正式日志。这种模式下，即使单个节点崩溃，也仅丢失该节点最后未提交的一个 Chunk，而不会导致全局日志丢失。

### 4.3 缓存失效 (Cache Invalidation)
*   本地 LRU 缓存条目附加 `Version` 标签。
*   每次从缓存读取数据时，比对本地 Version 与 `common.DB` 中的全局 Version。若版本落后，则立即执行 `Purge` 并重新加载。

## 5. 实施 Roadmap (Updated)
1.  **Infrastructure 层**: 在 `pkg/common` 中实现 `WithJitter(fn)`、`GetGlobalVersion(module)` 和 `NotifyCluster(event)` 等基础工具函数。
2.  **Phase 1 (Critical)**: 改造 `MMDBManager`。引入随机抖动重载和冷启动版本比对，消除 I/O 惊群效应。
3.  **Phase 2**: 改造 `TaskLogger`。实现基于分布式 KV 暂存与分块追加的流式日志系统。
4.  **Phase 3**: 完善基于租约的分布式 Cron 触发器，彻底解决多机部署下的定时任务重复执行隐患。
