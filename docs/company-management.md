# Company Management — 设计文档

## 概述
用"公司管理"替代原有的 Clusters 和 Messages 功能。
将 AI agent 集群视为一家虚拟公司来管理。

## 六大模块

### 1. 公司战略（Company Plan）
- 公司愿景、使命、当前目标
- 整体计划由管理层 agent 执行
- 目标可分解为子目标，分配到部门

### 2. 组织结构（Org Structure）
- 根据战略目标自动生成组织结构模板
- 树状层级：公司 → 部门 → 小组 → 岗位
- 每个岗位绑定一个 agent，管理：
  - 人设（persona）、背景（background）
  - KPI 指标、汇报对象
  - IM 联系方式（Telegram/Slack/webhook等）

### 3. KPI 考核（Performance Review）
- 考核周期管理（周报/双周/月度）
- 每个 agent 都有绩效考核
- 小组长汇总小组工作 → 管理层审阅反馈
- 考核指标：任务完成率、响应速度、质量评分等

### 4. 公司 IM（Communication）
- 根据组织结构自动组建群聊
- 全员群、部门群、小组群、1v1
- 支持消息路由到外部 IM（Telegram/Slack/webhook）

### 5. Workflow 管理
- 按岗位定义标准工作流
- 该岗位的 agent 须按 workflow 执行和反馈
- workflow 步骤、触发条件、产出要求

### 6. 治理工具（Governance）
- 公告系统
- 权限管理（谁能改组织结构、谁能发公告）
- 审计日志
- 紧急事件升级机制

## 数据模型

### company
| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | PK |
| name | string | 公司名 |
| vision | text | 愿景 |
| mission | text | 使命 |
| status | string | active/archived |

### company_goal
| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | PK |
| company_id | string | FK |
| title | string | 目标名 |
| description | text | 详细描述 |
| priority | string | high/medium/low |
| status | string | planning/active/completed |
| owner_position_id | string | 负责人岗位 |
| deadline | datetime | 截止日期 |
| parent_goal_id | string | 父目标(可分解) |

### department
| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | PK |
| company_id | string | FK |
| name | string | 部门名 |
| parent_id | string | 父部门(树状) |
| type | string | management/engineering/operations/... |
| description | text | 职责描述 |

### position
| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | PK |
| department_id | string | FK |
| title | string | 岗位名(CEO/CTO/Developer/...) |
| level | string | c-suite/director/manager/staff |
| reports_to | string | 汇报给哪个 position |
| responsibilities | text | 职责 |
| workflow_id | string | 关联的 workflow |

### agent_assignment
| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | PK |
| agent_id | string | FK → agent_instances |
| position_id | string | FK → position |
| persona | text | 人设描述 |
| background | text | 背景故事 |
| im_type | string | telegram/slack/webhook |
| im_config | json | IM 配置详情 |
| status | string | active/inactive |

### review_cycle
| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | PK |
| company_id | string | FK |
| name | string | 周期名(Weekly/Monthly) |
| frequency | string | weekly/biweekly/monthly |
| metrics | json | 考核指标列表 |
| report_template | text | 汇报模板 |

### kpi_review
| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | PK |
| agent_id | string | FK |
| cycle_id | string | FK → review_cycle |
| period | string | 2026-W07 / 2026-02 |
| metrics | json | 各指标得分 |
| score | float | 综合分 |
| self_report | text | agent自评 |
| reviewer_id | string | 评审人(manager agent_id) |
| feedback | text | 反馈 |
| status | string | pending/submitted/reviewed |

### im_group
| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | PK |
| name | string | 群名 |
| department_id | string | 关联部门 |
| type | string | all-hands/department/team/1v1 |
| members | json | agent_id 列表 |
| external_channel | json | 外部IM配置 |

### workflow
| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | PK |
| name | string | workflow名 |
| position_title | string | 适用岗位 |
| steps | json | 步骤列表 |
| triggers | json | 触发条件 |
| output_requirements | text | 产出要求 |

### governance_log
| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | PK |
| company_id | string | FK |
| action | string | 操作类型 |
| actor_id | string | 操作人 |
| detail | text | 详情 |
| timestamp | datetime | 时间 |
