# Clawfleet - 技术设计方案

## 1. 系统架构

```
                    ┌─────────────────────────┐
                    │   Web UI (Vue + Vite)   │
                    └───────────┬─────────────┘
                                │ HTTPS (SPA)
                    ┌───────────▼─────────────┐
                    │  Control Plane (Go)      │
                    │  clawfleet-server        │
                    │                          │
                    │  REST API  +  SPA static │
                    │  Scheduler + Task Queue  │
                    │  SQLite (single file)    │
                    └───────────┬─────────────┘
                                │ HTTPS JSON
                                │ /api/register
                                │ /api/heartbeat
                                │ /api/task/complete
                    ┌───────────▼─────────────┐
                    │  Node Agent (Go)         │
                    │  clawfleet-node          │
                    │                          │
                    │  Poll tasks / execute     │
                    │  Docker: create/stop/rm   │
                    └───────────┬─────────────┘
                                │ Docker
                    ┌───────────▼─────────────┐
                    │ Agent Containers         │
                    │ OpenClaw / ZeroClaw /    │
                    │ Nanobot (adapters)       │
                    └─────────────────────────┘
```

## 2. 核心组件

### 2.1 Control Plane (`clawfleet-server`)

单二进制，内嵌所有依赖，零外部服务。

| 模块 | 职责 |
|------|------|
| **API Server** | REST API + SPA 静态文件（web/dist） |
| **Node Manager** | 节点注册、心跳、状态管理 |
| **Agent Manager** | Agent 实例 CRUD、生命周期管理（异步 delete + task 回收容器） |
| **Scheduler** | 任务队列、定时任务（KPI auto review 等） |

**存储：**
- **SQLite** — 节点、agent、任务、公司/组织架构等元数据（当前版本单机足够）
- **本地文件系统** — 部署目录、web/dist、日志（按需）

### 2.2 Node Agent (`clawfleet-node`)

轻量级守护进程，安装在每个节点上。

| 功能 | 说明 |
|------|------|
| **注册** | 启动时向 Control Plane 注册，上报资源信息 |
| **心跳** | 每 15s 发送心跳 + 资源使用率 |
| **容器管理** | 通过 Docker API 创建/启停/删除 agent 容器 |
| **日志转发** | 收集容器日志，转发到 Control Plane |
| **任务执行** | 周期性拉取 tasks 并执行（deploy/stop/remove/restart/reload-persona） |

### 2.3 Agent Adapters

每种 Agent 框架一个适配器，定义：
- Docker 镜像 / 安装方式
- 启动命令和配置注入方式
- 健康检查方式
- 消息接口对接方式

```go
type AgentAdapter interface {
    Name() string
    // 返回容器配置
    ContainerConfig(instance AgentInstance) ContainerConfig
    // 健康检查
    HealthCheck(containerID string) (bool, error)
    // 向 agent 发送消息
    SendMessage(containerID string, msg Message) error
    // 从 agent 接收消息的回调
    OnMessage(containerID string, handler MessageHandler) error
}
```

MVP 实现三个：
- `OpenClawAdapter`
- `ZeroClawAdapter`
- `NanobotAdapter`

## 3. 通信协议

### 3.1 Control Plane ↔ Node Agent（当前实现）

HTTP/HTTPS JSON 请求（agent-key 保护），Node Agent 轮询任务：

- `POST /api/register` — 注册节点（使用 token/agent-key）
- `POST /api/heartbeat` — 心跳 + 资源（包含 node 版本信息）
- `GET  /api/tasks?node_id=...`（由 server 返回 pending tasks；当前实现为心跳响应携带或后续扩展）
- `POST /api/task/complete` — 上报任务执行结果

> 说明：当前版本不使用 gRPC/NATS/WS；后续如果需要更实时的推送/流式，可再演进协议。

### 3.2 Agent ↔ Agent 消息格式

```json
{
    "id": "msg_abc123",
    "from": "ceo-agent-01",
    "to": "developer-agent-01",    // 或 "*" 广播
    "type": "task_assign",
    "priority": "normal",
    "payload": {
        "task_id": "task_001",
        "title": "实现用户登录 API",
        "description": "...",
        "deadline": "2025-07-28T18:00:00Z"
    },
    "reply_to": null,
    "timestamp": "2025-07-27T10:00:00Z"
}
```

消息类型：
- `task_assign` / `task_result` / `task_update`
- `question` / `answer`
- `notification`
- `approval_request` / `approval_response`

## 4. 项目结构

```
clawfleet/
├── cmd/
│   ├── server/          # Control Plane 入口
│   │   └── main.go
│   └── agent/           # Node Agent 入口
│       └── main.go
├── internal/
│   ├── server/          # Control Plane 核心逻辑
│   │   ├── api/         # REST API handlers
│   │   ├── node/        # 节点管理
│   │   ├── agent/       # Agent 实例管理
│   │   ├── scheduler/   # 调度器
│   │   ├── comms/       # 消息路由
│   │   └── monitor/     # 监控
│   ├── agent/           # Node Agent 核心逻辑
│   ├── agent/           # 任务执行器 + runtime (docker/direct)
│   ├── adapter/         # Agent 适配器（OpenClaw/ZeroClaw/Nanobot）
│   ├── model/           # 数据模型
│   ├── server/          # API handlers + auth + scheduler
│   ├── store/           # 存储层 (SQLite)
│   ├── sync/            # persona/bot sync
│   ├── util/            # 通用工具
│   └── version/         # build info
├── web/                 # Web UI (Vue + Vite)
├── scripts/
│   └── deploy.sh        # 部署脚本
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## 5. 状态（实现情况概览）

> 本仓库已完成并在生产环境跑通的能力（非 roadmap）：

- [x] Node 注册/心跳/任务执行（deploy/stop/remove/restart）
- [x] Docker 容器管理（OpenClaw/ZeroClaw/Nanobot adapters）
- [x] Company/Org/Position/Assignment 管理
- [x] Telegram Bots（公司级配置 + Position 绑定 + 自动同步到 agent）
- [x] IM Groups / Workflows / KPI Review Cycles
- [x] Admin Credentials 修改（持久化到 SQLite）
- [x] 版本管理：`/api/version` + Settings 页展示 + node 上报版本

未来演进（可选）：实时推送、更多 IM 平台、分布式存储等。
- [ ] Agent 间消息收发
- [ ] 集群模板引擎
- [ ] Solo Corp 模板一键部署

### Phase 4: Web UI + 联调（Week 4）
- [ ] 基础 Web UI（节点列表、Agent 列表、创建集群）
- [ ] 端到端联调测试
- [ ] 安装脚本
- [ ] README + 文档
- [ ] 推送到 GitHub

## 6. 本地开发

```bash
# 启动 Control Plane
go run cmd/server/main.go

# 启动 Node Agent（本地测试，连接本地 server）
go run cmd/agent/main.go --server localhost:8080

# 构建
make build-all  # 输出 dist/bin/clawfleet-server + dist/bin/clawfleet-node + dist/bin/fleetctl
```
