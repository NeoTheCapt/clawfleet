# AgentCluster - 智能体集群管理平台 PRD

> Version: 0.1 Draft | Date: 2025-07-27 | Author: Brian & 小波

---

## 1. 背景与愿景

当前 AI Agent 生态中，各类智能体（OpenClaw、ZeroClaw、AutoGPT、CrewAI 等）都是单机部署、独立运行。缺乏一个统一的平台来：

- 在多台机器上批量部署和管理 agent
- 让多个 agent 协作形成"一人公司"
- 按模板快速生成专业化的 agent 团队

**AgentCluster** 的目标：成为 AI Agent 的 Kubernetes —— 一个分布式智能体编排与管理平台。

---

## 2. 核心概念

| 概念 | 说明 |
|------|------|
| **Control Plane** | 中心化控制平面，管理所有节点和智能体 |
| **Node** | 安装了 Agent Runtime 的物理/虚拟机器 |
| **Agent Runtime** | 节点上的 agent 执行环境（容器化隔离） |
| **Agent Instance** | 一个运行中的智能体实例 |
| **Cluster Template** | 预定义的智能体团队模板（如"一人公司"） |
| **Role** | 智能体在集群中的角色（Manager / Developer / Analyst 等） |
| **Channel** | 智能体之间的通信通道 |

---

## 3. 用户角色

| 角色 | 说明 |
|------|------|
| **平台管理员** | 管理节点、全局配置、资源分配 |
| **团队创建者** | 使用模板创建智能体团队，定义角色和任务 |
| **观察者** | 监控智能体运行状态和输出 |

---

## 4. 功能模块

### 4.1 节点管理（Node Management）

**目标：** 在不同机器上安装 agent runtime，自动连入控制平面。

- **一键安装脚本：** `curl -fsSL https://agentcluster.dev/install | bash`
  - 安装 Agent Runtime（轻量级守护进程）
  - 自动注册到 Control Plane
  - 上报节点资源信息（CPU / RAM / GPU / 磁盘）
- **节点生命周期：**
  - 注册 → 在线 → 忙碌 → 离线 → 移除
  - 心跳检测（30s 间隔）
  - 自动重连 & 断线告警
- **节点标签：** 支持给节点打标签（如 `gpu=true`, `region=sg`），用于调度

### 4.2 智能体运行时（Agent Runtime）

**目标：** 在节点上隔离运行各类 AI Agent。

- **容器化隔离：** 每个 agent instance 运行在独立容器中
  - 基于 Docker / Podman
  - 文件系统、网络、进程隔离
  - 资源限制（CPU / 内存 / 磁盘配额）
- **Agent 适配器（Adapter）：** 统一接口适配不同 agent 框架
  - OpenClaw Adapter
  - ZeroClaw Adapter
  - AutoGPT Adapter
  - CrewAI Adapter
  - 自定义 Adapter（插件化）
- **Agent 生命周期：**
  - 创建 → 初始化 → 运行中 → 暂停 → 终止
  - 支持热重启、配置热更新
  - 日志收集 & 持久化

### 4.3 集群模板（Cluster Templates）

**目标：** 按模板一键生成智能体团队。

- **内置模板：**
  - **一人公司（Solo Corp）：** CEO(统筹) + Developer + Analyst + Writer + QA
  - **研究团队：** Lead Researcher + Literature Review + Data Analysis + Writer
  - **开发团队：** Tech Lead + Frontend Dev + Backend Dev + DevOps + QA
  - **运营团队：** PM + Content Creator + Data Analyst + Customer Support
- **模板定义格式（YAML）：**

```yaml
name: solo-corp
version: 1.0
description: "一人公司 - 全功能智能体团队"

roles:
  - id: ceo
    name: CEO / 总指挥
    agent_type: openclaw
    model: claude-sonnet-4-20250514
    system_prompt_file: prompts/ceo.md
    capabilities: [task_delegation, progress_tracking, decision_making]
    can_manage: [developer, analyst, writer, qa]

  - id: developer
    name: 开发工程师
    agent_type: openclaw
    model: claude-sonnet-4-20250514
    system_prompt_file: prompts/developer.md
    capabilities: [coding, debugging, deployment]
    tools: [github, exec, browser]

  - id: analyst
    name: 数据分析师
    agent_type: openclaw
    model: claude-sonnet-4-20250514
    system_prompt_file: prompts/analyst.md
    capabilities: [data_analysis, visualization, reporting]
    tools: [exec, browser, web_search]

  - id: writer
    name: 内容创作者
    agent_type: zeroclaw
    model: gpt-4o
    system_prompt_file: prompts/writer.md
    capabilities: [writing, editing, seo]

  - id: qa
    name: 质量保证
    agent_type: openclaw
    model: claude-sonnet-4-20250514
    system_prompt_file: prompts/qa.md
    capabilities: [testing, review, validation]

communication:
  topology: hierarchical  # hierarchical | mesh | ring
  channels:
    - name: management
      members: [ceo]
      type: broadcast  # ceo 广播给所有人
    - name: dev-qa
      members: [developer, qa]
      type: peer
    - name: all-hands
      members: [ceo, developer, analyst, writer, qa]
      type: group

scheduling:
  node_affinity:
    developer: { label: "gpu=true" }  # 开发者优先 GPU 节点
  resource_limits:
    default:
      cpu: "2"
      memory: "4Gi"
```

- **自定义模板：** 用户可创建、分享、导入模板
- **模板市场：** 类似 ClawHub，社区共享模板

### 4.4 智能体通信（Agent Communication）

**目标：** 让智能体之间高效协作。

- **现状：** 当前版本不提供独立的消息队列（NATS/Redis Streams 未接入）。
- **当前可用通信入口：**
  - OpenClaw 自身的 channel（如 Telegram），由 Clawfleet 负责部署与 token 注入。
- **未来演进：** 可引入消息队列实现 Agent↔Agent 可靠投递。
- **通信模式：**
  - **直接消息：** A → B
  - **广播：** A → [B, C, D]
  - **任务委派：** Manager 分配任务给 Worker，Worker 返回结果
  - **事件订阅：** Agent 订阅特定事件类型
- **消息类型：**
  - `task_assign` — 分配任务
  - `task_result` — 返回结果
  - `status_update` — 状态更新
  - `question` — 询问
  - `approval_request` — 审批请求
  - `notification` — 通知
- **人类干预点（Human-in-the-Loop）：**
  - 关键决策可路由给人类审批
  - 人类可随时介入任何通信通道
  - 支持设置自动化级别（全自动 / 半自动 / 手动审批）

### 4.5 任务编排（Task Orchestration）

**目标：** 支持复杂的多 agent 工作流。

- **任务定义：**
  - 自然语言描述目标
  - Manager Agent 自动拆解为子任务
  - 支持手动定义任务 DAG
- **执行模式：**
  - **串行：** A 完成 → B 开始
  - **并行：** A 和 B 同时执行
  - **条件分支：** if A.result == X then B else C
  - **循环：** 迭代直到满足条件
- **进度追踪：**
  - 任务看板（Kanban 视图）
  - 实时日志流
  - 关键里程碑通知

### 4.6 监控与可观测性（Observability）

- **Dashboard：**
  - 节点状态总览
  - Agent 实例列表与状态
  - 资源使用率（CPU / 内存 / API 调用量 / Token 消耗）
  - 任务执行进度
- **日志：**
  - 集中式日志收集
  - 按 agent / 任务 / 时间筛选
  - Agent 间对话历史
- **告警：**
  - 节点离线
  - Agent 异常退出
  - 资源超限
  - 任务超时
  - Token/API 预算告警
- **成本追踪：**
  - 按 agent / 任务 / 模型统计 token 用量和费用
  - 预算上限设置

### 4.7 控制平面 Web UI

- **节点管理页：** 查看/添加/移除节点
- **Agent 管理页：** 创建/启停/配置 agent instances
- **集群管理页：** 从模板创建团队，管理团队生命周期
- **任务中心：** 创建任务，查看进度，人工干预
- **模板编辑器：** 可视化编辑集群模板
- **设置：** API Keys 管理、全局配置、用户权限

---

## 5. 技术架构

```
┌─────────────────────────────────────────────┐
│              Web UI (Vue + Vite SPA)         │
└──────────────────────┬──────────────────────┘
                       │
┌──────────────────────▼──────────────────────┐
│            Control Plane API                 │
│  ┌──────────┐ ┌──────────┐ ┌──────────────┐ │
│  │ Node Mgr │ │Agent Mgr │ │ Task Engine  │ │
│  └──────────┘ └──────────┘ └──────────────┘ │
│  ┌──────────┐ ┌──────────┐ ┌──────────────┐ │
│  │Template  │ │ Comms    │ │  Monitoring  │ │
│  │ Engine   │ │ Broker   │ │  & Logging   │ │
│  └──────────┘ └──────────┘ └──────────────┘ │
│                                              │
│  Storage: SQLite (single file)               │
│  Message Bus: (none in current version)      │
└──────┬───────────────┬──────────────┬────────┘
       │               │              │
  ┌────▼────┐    ┌─────▼────┐   ┌────▼────┐
  │ Node A  │    │ Node B   │   │ Node C  │
  │┌───────┐│    │┌───────┐ │   │┌───────┐│
  ││Runtime││    ││Runtime│ │   ││Runtime││
  │├───────┤│    │├───────┤ │   │├───────┤│
  ││Agent 1││    ││Agent 3│ │   ││Agent 5││
  ││Agent 2││    ││Agent 4│ │   ││Agent 6││
  │└───────┘│    │└───────┘ │   │└───────┘│
  └─────────┘    └──────────┘   └─────────┘
```

### 技术选型建议

| 层 | 选型 | 理由 |
|----|------|------|
| Control Plane API | Go / Rust | 高性能、低资源占用 |
| Web UI | Vue + Vite + Tailwind | SPA 管理后台，构建简单 |
| 数据库 | SQLite | 当前版本单机/轻量部署优先 |
| 消息总线 | （无） | 当前版本采用 HTTP task polling，不依赖 MQ |
| 容器运行时 | Docker API | 通用、生态好 |
| Agent Runtime | Node.js daemon | 与 OpenClaw 生态一致 |
| 对象存储 | MinIO / S3 | 日志、文件、工件 |

---

## 6. MVP 范围（Phase 1）

**目标：** 3-4 周内跑通核心流程

### P0 - 必须有（当前实现已覆盖）
- [x] Control Plane API（CRUD + 任务队列）
- [x] Node Agent 安装脚本 + 自动注册（/api/install.sh + /dl/fleet-node）
- [x] 节点心跳 & 状态管理
- [x] 在节点上创建/启停/删除 OpenClaw/ZeroClaw/Nanobot agent 容器
- [x] 基础 Web UI（Nodes/Agents/Companies/Settings）

### P1 - 应该有（部分已实现）
- [x] 多 agent 框架适配（OpenClaw/ZeroClaw/Nanobot）
- [x] Company/Org/Position/Assignment 管理
- [x] Telegram Bots（公司级 + Position 绑定 + 自动同步到 agent）
- [x] IM Groups / Workflows / KPI Review Cycles
- [ ] 任务编排引擎（串行/并行）
- [ ] Human-in-the-Loop 审批
- [ ] 成本/Token 追踪
- [ ] 日志集中查看

### P2 - 可以有
- [ ] 模板市场
- [ ] 可视化模板编辑器
- [ ] 高级调度（GPU 亲和、地域感知）
- [ ] 多租户 & 权限管理
- [ ] 自定义 Adapter SDK

---

## 7. 已确认决策

| 决策项 | 结论 |
|--------|------|
| **项目名称** | Clawfleet |
| **GitHub 仓库** | https://github.com/NeoTheCapt/clawfleet |
| **通信协议** | MVP 自定义，消息格式参考 A2A 概念，后续兼容 A2A 标准 |
| **优先适配 Agent** | OpenClaw, ZeroClaw, Nanobot |
| **部署模式** | 自托管优先 |
| **Control Plane 语言** | Go |
| **Web UI** | Vue + Vite + Tailwind |

## 8. 开放问题

1. **安全模型：** 节点间通信加密方案？Agent 容器的安全边界？
2. **状态持久化：** Agent 重启后如何恢复上下文？
3. **成本控制：** 如何防止 agent 团队失控烧钱？硬限制 vs 软限制？

---

## 8. 成功指标

- 5 分钟内完成新节点接入
- 1 分钟内从模板创建一个 5-agent 团队
- Agent 间消息延迟 < 500ms
- 单节点支持 10+ 并发 agent instance
- 系统可用性 > 99.5%

---

_This is a living document. 随时更新。_
