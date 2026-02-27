# 中心化人设背景件管理机制设计

## 🎯 设计目标

1. **中心化管理**：在 FleetCP 创建 Position 时生成人设背景
2. **智能同步**：分配智能体时自动同步人设背景到智能体
3. **动态更新**：修改人设背景时自动更新已分配的智能体
4. **模板复用**：支持人设模板，可复用标准人设配置

## 🏗️ 架构设计

### 数据模型
```go
// 1. 人设模板 (PersonaTemplate) - 中心化存储
type PersonaTemplate struct {
    ID           string    `json:"id"`
    Name         string    `json:"name"`           // 模板名称
    Description  string    `json:"description"`    // 模板描述
    SystemPrompt string    `json:"system_prompt"`  // 系统提示词
    Background   string    `json:"background"`     // 背景故事
    Tags         []string  `json:"tags"`           // 标签分类
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}

// 2. Position 增强
type Position struct {
    ID                string        `json:"id"`
    Title             string        `json:"title"`
    // ... 其他字段
    SystemPrompt      string        `json:"system_prompt,omitempty"`       // 自定义系统提示
    Background        string        `json:"background,omitempty"`          // 自定义背景故事
    PersonaTemplateID string        `json:"persona_template_id,omitempty"` // 引用的人设模板ID
    // ... 其他字段
}

// 3. AgentInstance 增强
type AgentInstance struct {
    ID      string      `json:"id"`
    // ... 其他字段
    Config  AgentConfig `json:"config"`
    // ... 其他字段
}

type AgentConfig struct {
    SystemPrompt string `json:"system_prompt"`  // 从 Position 同步
    PersonaName  string `json:"persona_name"`   // 角色名称
    Description  string `json:"description"`    // 背景故事
    // ... 其他字段
}
```

## 🔄 工作流程

### 1. Position 创建/更新时的人设配置
```
用户创建 Position
    ↓
选择人设配置方式：
    ├── 使用现有模板 (PersonaTemplateID)
    ├── 自定义系统提示 + 背景故事
    └── 使用模板 + 自定义覆盖
    ↓
保存到数据库
    ↓
触发人设验证（可选）
```

### 2. 分配智能体时的人设同步
```
POST /api/companies/{companyId}/assignments
    ↓
解析请求参数：
    - position_id: 必需
    - agent_id: 可选（使用现有智能体）
    - node_id: 可选（选择节点）
    ↓
获取 Position 详情
    ↓
计算有效人设配置：
    ├── 如果 Position 有自定义字段 → 使用自定义
    ├── 如果 Position 有模板ID → 从模板获取
    └── 如果都没有 → 使用默认模板（可选）
    ↓
创建/配置智能体：
    ├── 新智能体：创建时注入人设配置
    └── 现有智能体：更新人设配置
    ↓
启动智能体容器
    ↓
返回 Assignment 记录
```

### 3. 人设更新时的同步机制
```
PATCH /api/positions/{positionId}
    ↓
更新 Position 人设配置
    ↓
查找所有相关的 Assignments
    ↓
为每个已分配的智能体：
    ├── 如果智能体在线 → 发送配置更新消息
    ├── 如果智能体离线 → 标记为待更新
    └── 记录更新历史
    ↓
返回同步结果
```

## 📡 API 端点

### 人设模板管理
```
GET    /api/persona-templates              # 列表
POST   /api/persona-templates              # 创建
GET    /api/persona-templates/{id}         # 详情
PUT    /api/persona-templates/{id}         # 更新
DELETE /api/persona-templates/{id}         # 删除
```

### 人设同步操作
```
POST   /api/positions/{id}/sync-persona    # 同步到所有已分配智能体
POST   /api/agents/{id}/apply-persona      # 应用特定人设
GET    /api/positions/{id}/persona-status  # 查看人设同步状态
```

### Assignment 增强
```
POST   /api/companies/{companyId}/assignments
增加参数：
{
    "position_id": "pos_xxx",
    "agent_id": "agent_xxx",        # 可选：使用现有智能体
    "node_id": "node_xxx",          # 可选：选择节点
    "agent_type": "openclaw",       # 可选：智能体类型
    "deploy_mode": "docker",        # 可选：部署模式
    "skip_persona_sync": false      # 可选：跳过人设同步
}
```

## 💾 存储设计

### 数据库表
```sql
-- 人设模板表
CREATE TABLE persona_templates (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT DEFAULT '',
    system_prompt TEXT DEFAULT '',
    background TEXT DEFAULT '',
    tags TEXT DEFAULT '[]',  -- JSON 数组
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Position 增强字段
ALTER TABLE positions ADD COLUMN persona_template_id TEXT DEFAULT '';

-- 人设同步历史表
CREATE TABLE persona_sync_history (
    id TEXT PRIMARY KEY,
    position_id TEXT NOT NULL,
    agent_id TEXT NOT NULL,
    action TEXT NOT NULL,  -- 'create', 'update', 'sync'
    old_system_prompt TEXT DEFAULT '',
    new_system_prompt TEXT DEFAULT '',
    old_background TEXT DEFAULT '',
    new_background TEXT DEFAULT '',
    status TEXT NOT NULL,  -- 'pending', 'success', 'failed'
    error TEXT DEFAULT '',
    synced_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

## 🔧 实现细节

### 1. 人设配置解析器
```go
// 获取 Position 的有效人设配置
func GetEffectivePersona(positionID string) (systemPrompt, background string, err error) {
    // 1. 获取 Position
    // 2. 检查自定义字段
    // 3. 检查模板引用
    // 4. 返回有效配置
}
```

### 2. 智能体配置注入器
```go
// 注入人设配置到智能体
func InjectPersonaToAgent(agentID string, systemPrompt, background string) error {
    // 1. 获取智能体配置
    // 2. 更新系统提示词
    // 3. 更新背景故事
    // 4. 保存配置
    // 5. 触发智能体重载（如果运行中）
}
```

### 3. 人设同步器
```go
// 同步 Position 人设到所有已分配智能体
func SyncPersonaToAllAgents(positionID string) error {
    // 1. 获取 Position 的有效人设
    // 2. 查找所有相关的 Assignments
    // 3. 为每个智能体同步人设
    // 4. 记录同步历史
}
```

## 🚀 部署策略

### 1. 智能体配置存储
- **文件方式**：更新智能体的 `SOUL.md`、`IDENTITY.md` 等文件
- **环境变量**：通过环境变量传递配置
- **配置文件**：更新智能体的配置文件
- **API注入**：通过智能体 API 动态更新

### 2. 配置重载机制
- **容器重启**：重新部署容器应用新配置
- **热重载**：发送信号让智能体重载配置
- **动态更新**：通过智能体 API 更新运行中配置

## 📊 监控与日志

### 1. 监控指标
- 人设同步成功率
- 同步延迟统计
- 配置更新频率
- 模板使用情况

### 2. 日志记录
- 人设配置变更日志
- 同步操作日志
- 错误日志和异常处理
- 审计日志

## 🔐 安全性考虑

1. **配置验证**：验证系统提示词的安全性
2. **访问控制**：限制人设模板的创建和修改权限
3. **审计跟踪**：记录所有人设变更操作
4. **备份恢复**：定期备份人设模板数据

## 🎨 用户体验

### Web UI 功能
1. **人设模板库**：浏览、搜索、选择模板
2. **模板编辑器**：可视化编辑人设配置
3. **预览功能**：预览人设效果
4. **同步状态**：查看人设同步状态和历史
5. **批量操作**：批量同步人设配置

### 模板推荐系统
- 基于职位类型推荐模板
- 热门模板排行榜
- 个性化模板推荐

## 📈 扩展性设计

### 1. 多级模板继承
- 基础模板 → 部门模板 → 职位模板
- 支持模板继承和覆盖

### 2. 变量系统
- 支持模板变量：`{{position.title}}`、`{{company.name}}`
- 动态替换变量值

### 3. 版本管理
- 模板版本控制
- 历史版本回滚
- 变更对比功能

## 🔄 与其他系统的集成

### 1. 与 OpenClaw 集成
- 通过 Agent API 注入配置
- 支持 OpenClaw 的 SOUL.md 格式

### 2. 与部署系统集成
- 在部署流程中注入人设配置
- 支持容器镜像构建时预配置

### 3. 与监控系统集成
- 人设配置变更监控
- 同步状态告警

## 📋 实施计划

### Phase 1: 基础功能
1. 人设模板 CRUD API
2. Position 人设配置存储
3. 分配智能体时自动注入人设

### Phase 2: 同步机制
1. 人设同步 API
2. 同步历史记录
3. 错误处理和重试

### Phase 3: 高级功能
1. 模板变量系统
2. 版本管理
3. Web UI 集成

### Phase 4: 优化扩展
1. 性能优化
2. 监控告警
3. 高级模板功能

## 🎯 成功指标

1. **配置准确性**：人设配置正确率 > 99.9%
2. **同步延迟**：同步操作平均延迟 < 1s
3. **系统可用性**：人设管理服务可用性 > 99.5%
4. **用户满意度**：模板使用率 > 80%

---

*本文档为技术设计文档，实施时需根据实际情况调整。*