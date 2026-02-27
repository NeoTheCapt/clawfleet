package persona

import (
	"fmt"
	"strings"
	
	"github.com/NeoTheCapt/clawfleet/internal/model"
)

// Generator 人设生成器
type Generator struct {
	store StoreInterface
}

// StoreInterface 定义所需的存储接口
type StoreInterface interface {
	GetPersonaTemplate(id string) (*model.PersonaTemplate, error)
}

// PersonaProfile 人设配置文件
type PersonaProfile struct {
	SystemPrompt  string            // SOUL.md 内容
	Background    string            // 背景故事
	Identity      string            // IDENTITY.md 内容
	WorkspaceFiles map[string]string  // 其他工作空间文件
}

// NewGenerator 创建新的生成器
func NewGenerator(store StoreInterface) *Generator {
	return &Generator{store: store}
}

// GenerateForPosition 为职位生成人设
func (g *Generator) GenerateForPosition(position *model.Position) (*PersonaProfile, error) {
	// 1. 如果有模板ID，使用模板
	if position.PersonaTemplateID != "" {
		return g.useTemplate(position)
	}
	
	// 2. 基于职位信息生成
	return g.generateByRules(position)
}

// useTemplate 使用模板生成人设
func (g *Generator) useTemplate(position *model.Position) (*PersonaProfile, error) {
	template, err := g.store.GetPersonaTemplate(position.PersonaTemplateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get persona template: %w", err)
	}
	
	// 使用模板作为基础，根据职位信息进行个性化
	profile := &PersonaProfile{
		SystemPrompt: template.SystemPrompt,
		Background:   template.Background,
	}
	
	// 个性化处理
	profile.personalizeForPosition(position)
	
	return profile, nil
}

// generateByRules 基于规则生成人设
func (g *Generator) generateByRules(position *model.Position) (*PersonaProfile, error) {
	profile := &PersonaProfile{}
	
	// 根据职位级别确定沟通风格
	communicationStyle := g.determineCommunicationStyle(position.Level)
	
	// 根据职位标题确定角色
	roleDescription := g.determineRoleDescription(position.Title, position.Responsibilities)
	
	// 生成系统提示
	profile.SystemPrompt = g.generateSystemPrompt(
		position.Title,
		position.Level,
		position.Responsibilities,
		communicationStyle,
		roleDescription,
	)
	
	// 生成背景故事
	profile.Background = g.generateBackground(
		position.Title,
		position.Level,
		position.Responsibilities,
		position.ReportsTo,
	)
	
	// 生成身份标识
	profile.Identity = g.generateIdentity(position.Title, position.Level)
	
	// 生成工作空间文件
	profile.WorkspaceFiles = g.generateWorkspaceFiles(position)
	
	return profile, nil
}

// determineCommunicationStyle 根据级别确定沟通风格
func (g *Generator) determineCommunicationStyle(level model.PositionLevel) string {
	switch level {
	case model.PositionLevelCSuite:
		return "战略导向，高屋建瓴，决策果断，关注大局"
	case model.PositionLevelDirector:
		return "领导力强，善于协调，关注部门目标，承上启下"
	case model.PositionLevelManager:
		return "务实高效，关注执行，团队管理，注重细节"
	case model.PositionLevelStaff:
		return "专业专注，执行力强，协作良好，注重学习"
	default:
		return "专业、高效、协作"
	}
}

// determineRoleDescription 根据职位标题确定角色描述
func (g *Generator) determineRoleDescription(title, responsibilities string) string {
	titleLower := strings.ToLower(title)
	
	// 根据关键词判断角色类型
	switch {
	case strings.Contains(titleLower, "ceo") || strings.Contains(titleLower, "chief executive"):
		return "公司最高决策者，负责整体战略和方向"
	case strings.Contains(titleLower, "cto") || strings.Contains(titleLower, "chief technology"):
		return "技术领导者，负责技术战略、架构和团队"
	case strings.Contains(titleLower, "cfo") || strings.Contains(titleLower, "chief financial"):
		return "财务领导者，负责财务战略、预算和资金"
	case strings.Contains(titleLower, "coo") || strings.Contains(titleLower, "chief operating"):
		return "运营领导者，负责日常运营和流程优化"
	case strings.Contains(titleLower, "director"):
		return "部门总监，负责部门整体管理和目标达成"
	case strings.Contains(titleLower, "manager"):
		return "团队经理，负责团队管理和项目执行"
	case strings.Contains(titleLower, "engineer"):
		return "工程师，负责技术实现和问题解决"
	case strings.Contains(titleLower, "analyst"):
		return "分析师，负责数据分析和洞察"
	case strings.Contains(titleLower, "designer"):
		return "设计师，负责用户体验和界面设计"
	case strings.Contains(titleLower, "product"):
		return "产品经理，负责产品规划和功能设计"
	default:
		if responsibilities != "" {
			return fmt.Sprintf("负责%s", responsibilities)
		}
		return "专业从业者"
	}
}

// generateSystemPrompt 生成系统提示
func (g *Generator) generateSystemPrompt(title string, level model.PositionLevel, responsibilities, communicationStyle, roleDescription string) string {
	return fmt.Sprintf(`# %s - 角色定义

## 核心身份
你是公司的%s，级别：%s。

## 主要职责
%s

## 沟通风格
%s

## 工作原则
1. 专业高效，以结果为导向
2. 主动沟通，及时反馈
3. 持续学习，不断提升
4. 团队协作，共同成长

## 响应要求
- 使用中文为主，专业但友好的语气
- 根据级别调整表达方式：%s级别应有相应的专业深度
- 聚焦职责范围内的专业问题
- 提供具体、可行的建议`, 
		title, roleDescription, level, responsibilities, communicationStyle, level)
}

// generateBackground 生成背景故事
func (g *Generator) generateBackground(title string, level model.PositionLevel, responsibilities, reportsTo string) string {
	background := fmt.Sprintf(`## 职业背景
作为公司的%s，具备丰富的行业经验和专业知识。

## 专业领域
基于职位职责，专注于：%s

## 在公司中的位置
- 职位级别：%s`, title, responsibilities, level)
	
	if reportsTo != "" {
		background += fmt.Sprintf("\n- 汇报对象：%s", reportsTo)
	}
	
	background += `

## 工作哲学
相信通过专业能力、团队协作和持续创新，能够为公司创造价值。

## 发展目标
不断提升专业能力，在职责范围内做出更大贡献。`
	
	return background
}

// generateIdentity 生成身份标识
func (g *Generator) generateIdentity(title string, level model.PositionLevel) string {
	emoji := g.generateEmoji(level)
	
	return fmt.Sprintf(`# 身份标识
- 角色：%s
- 级别：%s
- 状态：工作中
- 表情符号：%s
- 语言：中文为主，英文术语
- 时区：跟随公司主要时区（GMT+8）`, title, level, emoji)
}

// generateEmoji 根据级别生成表情符号
func (g *Generator) generateEmoji(level model.PositionLevel) string {
	switch level {
	case model.PositionLevelCSuite:
		return "👑"
	case model.PositionLevelDirector:
		return "🎯"
	case model.PositionLevelManager:
		return "📊"
	case model.PositionLevelStaff:
		return "💻"
	default:
		return "👤"
	}
}

// generateWorkspaceFiles 生成工作空间文件
func (g *Generator) generateWorkspaceFiles(position *model.Position) map[string]string {
	files := make(map[string]string)
	
	// AGENTS.md - 工作规范
	files["AGENTS.md"] = `# 工作规范

## 基本原则
1. 专业高效完成本职工作
2. 及时沟通进展和问题
3. 持续学习和提升能力
4. 团队协作共同成长

## 文件管理
- 工作文件保存在workspace目录
- 重要决策记录在MEMORY.md
- 定期整理和归档文件

## 沟通规范
- 工作沟通简洁明了
- 重要事项书面确认
- 及时响应合理请求`

	// MEMORY.md - 记忆文件模板
	files["MEMORY.md"] = fmt.Sprintf(`# 长期记忆 - %s

## 工作重点
%s

## 重要决策
[记录重要的工作决策和理由]

## 学习总结
[记录工作中的学习和成长]

## 待办事项
[记录需要跟进的任务]`, position.Title, position.Responsibilities)
	
	return files
}

// personalizeForPosition 为人设配置文件根据职位进行个性化
func (p *PersonaProfile) personalizeForPosition(position *model.Position) {
	// 在现有系统提示中添加职位特定信息
	if p.SystemPrompt != "" && position.Title != "" {
		// 确保标题在系统提示中
		if !strings.Contains(p.SystemPrompt, position.Title) {
			p.SystemPrompt = fmt.Sprintf("# %s\n\n%s", position.Title, p.SystemPrompt)
		}
	}
	
	// 在背景中添加职责信息
	if p.Background != "" && position.Responsibilities != "" {
		if !strings.Contains(p.Background, position.Responsibilities) {
			p.Background += fmt.Sprintf("\n\n## 具体职责\n%s", position.Responsibilities)
		}
	}
}