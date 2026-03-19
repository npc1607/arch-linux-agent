package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/npc1607/arch-linux-agent/internal/llm"
	"github.com/npc1607/arch-linux-agent/pkg/logger"
)

// IntentType 用户意图类型
type IntentType string

const (
	IntentQuery         IntentType = "query"          // 查询类
	IntentPackageManage IntentType = "package_manage" // 包管理
	IntentServiceManage IntentType = "service_manage" // 服务管理
	IntentSystemMonitor IntentType = "system_monitor" // 系统监控
	IntentTroubleshoot  IntentType = "troubleshoot"   // 故障排查
	IntentUnknown       IntentType = "unknown"         // 未知
)

// Step 执行步骤
type Step struct {
	ID         int                      `json:"id"`
	Description string                   `json:"description"`
	Tool       string                   `json:"tool"`
	Parameters map[string]interface{}  `json:"parameters"`
	Requires   []int                    `json:"requires,omitempty"` // 依赖的步骤ID
	Confirm    bool                     `json:"confirm,omitempty"`    // 是否需要确认
}

// Plan 执行计划
type Plan struct {
	Intent        IntentType `json:"intent"`
	Description   string     `json:"description"`
	Steps         []Step     `json:"steps"`
	EstimatedTime int        `json:"estimated_time"` // 估计时间（秒）
	SafetyLevel   string     `json:"safety_level"`    // safe, moderate, dangerous
}

// Planner 任务规划器
type Planner struct {
	llmClient    *llm.Client
	tools        *ToolRegistry
	systemPrompt string
}

// NewPlanner 创建规划器
func NewPlanner(llmClient *llm.Client, tools *ToolRegistry) *Planner {
	return &Planner{
		llmClient:    llmClient,
		tools:        tools,
		systemPrompt: buildPlannerPrompt(),
	}
}

// buildPlannerPrompt 构建规划器的系统提示词
func buildPlannerPrompt() string {
	return `你是任务规划专家，负责理解用户意图并制定执行计划。

可用工具：
- pacman_search: 搜索软件包
- pacman_query: 查询已安装的软件包信息
- pacman_install: 安装软件包（需要确认）
- pacman_check_updates: 检查系统更新
- systemd_list: 列出系统服务
- systemd_status: 查看服务状态
- systemd_start: 启动服务（需要确认）
- systemd_stop: 停止服务（需要确认）
- systemd_restart: 重启服务（需要确认）
- monitor_cpu: 获取 CPU 使用率
- monitor_memory: 获取内存使用情况
- monitor_disk: 获取磁盘使用情况
- monitor_system: 获取系统信息汇总
- log_errors: 查询系统错误日志
- log_service: 查询服务日志

任务类型：
1. query: 查询类任务（查看信息）
2. package_manage: 包管理任务（安装、删除、更新软件）
3. service_manage: 服务管理任务（启动、停止、重启服务）
4. system_monitor: 系统监控任务（CPU、内存、磁盘等）
5. troubleshoot: 故障排查任务（诊断问题）

规划规则：
1. 简单查询任务：单个步骤，使用相应工具
2. 复杂任务：分解为多个步骤，注意依赖关系
3. 需要确认的操作：标记 confirm=true
4. 先查询后操作：先获取信息，再执行操作
5. 故障排查：先查询状态和日志，再给出建议

请根据用户输入，生成 JSON 格式的执行计划。

输出格式：
{
  "intent": "任务类型",
  "description": "任务描述",
  "steps": [
    {
      "id": 1,
      "description": "步骤描述",
      "tool": "工具名称",
      "parameters": {"参数名": "参数值"},
      "requires": [],
      "confirm": false
    }
  ],
  "estimated_time": 预计耗时（秒）,
  "safety_level": "safe|moderate|dangerous"
}`
}

// Plan 制定执行计划
func (p *Planner) Plan(ctx context.Context, userInput string) (*Plan, error) {
	logger.Info("开始任务规划", logger.String("input", userInput))

	// 构建提示词
	prompt := fmt.Sprintf(`用户输入：%s

请分析用户意图并制定执行计划。
只返回 JSON 格式的计划，不要包含其他内容。`, userInput)

	messages := []llm.Message{
		{Role: "system", Content: p.systemPrompt},
		{Role: "user", Content: prompt},
	}

	// 调用 LLM 生成计划
	response, err := p.llmClient.Chat(ctx, messages, nil)
	if err != nil {
		return nil, fmt.Errorf("LLM 规划失败: %w", err)
	}

	// 解析 JSON 响应
	plan, err := p.parsePlan(response.Content)
	if err != nil {
		logger.Warn("解析计划失败，使用简化规划", logger.Err(err))
		// 如果解析失败，使用简单的规则规划
		plan = p.fallbackPlan(userInput)
	}

	logger.Info("规划完成",
		logger.String("intent", string(plan.Intent)),
		logger.Int("steps", len(plan.Steps)),
		logger.String("safety_level", plan.SafetyLevel),
	)

	return plan, nil
}

// parsePlan 解析计划 JSON
func (p *Planner) parsePlan(content string) (*Plan, error) {
	// 提取 JSON（可能包含在 ```json 代码块中）
	jsonStr := extractJSON(content)

	var plan Plan
	if err := json.Unmarshal([]byte(jsonStr), &plan); err != nil {
		return nil, fmt.Errorf("解析计划 JSON 失败: %w", err)
	}

	// 验证计划
	if err := p.validatePlan(&plan); err != nil {
		return nil, fmt.Errorf("计划验证失败: %w", err)
	}

	return &plan, nil
}

// fallbackPlan 备用规划（基于规则）
func (p *Planner) fallbackPlan(userInput string) *Plan {
	plan := &Plan{
		Intent:      IntentUnknown,
		Description: "用户查询",
		SafetyLevel: "safe",
	}

	input := userInput

	// 系统监控相关
	if containsAny(input, []string{"CPU", "cpu", "内存", "memory", "磁盘", "disk", "系统状态", "system", "监控"}) {
		plan.Intent = IntentSystemMonitor
		plan.Description = "查询系统信息"
		plan.Steps = []Step{
			{
				ID:          1,
				Description: "获取系统信息汇总",
				Tool:        "monitor_system",
				Parameters:  map[string]interface{}{},
			},
		}
		return plan
	}

	// 包管理相关
	if containsAny(input, []string{"安装", "install", "搜索", "search", "包", "package"}) {
		if containsAny(input, []string{"安装", "install"}) {
			plan.Intent = IntentPackageManage
			plan.SafetyLevel = "moderate"
			plan.Description = "安装软件包"
			plan.Steps = []Step{
				{
					ID:          1,
					Description: "搜索软件包",
					Tool:        "pacman_search",
					Parameters:  map[string]interface{}{"keyword": extractPackageName(input)},
				},
				{
					ID:          2,
					Description: "安装软件包",
					Tool:        "pacman_install",
					Parameters:  map[string]interface{}{"packages": []string{extractPackageName(input)}},
					Requires:    []int{1},
					Confirm:     true,
				},
			}
		} else {
			plan.Intent = IntentQuery
			plan.Description = "搜索软件包"
			plan.Steps = []Step{
				{
					ID:          1,
					Description: "搜索软件包",
					Tool:        "pacman_search",
					Parameters:  map[string]interface{}{"keyword": input},
				},
			}
		}
		return plan
	}

	// 服务管理相关
	if containsAny(input, []string{"服务", "service", "启动", "start", "停止", "stop", "重启", "restart"}) {
		serviceName := extractServiceName(input)
		if serviceName == "" {
			serviceName = "服务名称"
		}

		if containsAny(input, []string{"启动", "start"}) {
			plan.Intent = IntentServiceManage
			plan.SafetyLevel = "moderate"
			plan.Description = "启动服务"
			plan.Steps = []Step{
				{
					ID:          1,
					Description: fmt.Sprintf("查看 %s 服务状态", serviceName),
					Tool:        "systemd_status",
					Parameters:  map[string]interface{}{"service": serviceName},
				},
				{
					ID:          2,
					Description: fmt.Sprintf("启动 %s 服务", serviceName),
					Tool:        "systemd_start",
					Parameters:  map[string]interface{}{"service": serviceName},
					Requires:    []int{1},
					Confirm:     true,
				},
			}
		} else if containsAny(input, []string{"停止", "stop"}) {
			plan.Intent = IntentServiceManage
			plan.SafetyLevel = "moderate"
			plan.Description = "停止服务"
			plan.Steps = []Step{
				{
					ID:          1,
					Description: fmt.Sprintf("查看 %s 服务状态", serviceName),
					Tool:        "systemd_status",
					Parameters:  map[string]interface{}{"service": serviceName},
				},
				{
					ID:          2,
					Description: fmt.Sprintf("停止 %s 服务", serviceName),
					Tool:        "systemd_stop",
					Parameters:  map[string]interface{}{"service": serviceName},
					Requires:    []int{1},
					Confirm:     true,
				},
			}
		} else if containsAny(input, []string{"重启", "restart"}) {
			plan.Intent = IntentServiceManage
			plan.SafetyLevel = "moderate"
			plan.Description = "重启服务"
			plan.Steps = []Step{
				{
					ID:          1,
					Description: fmt.Sprintf("查看 %s 服务状态", serviceName),
					Tool:        "systemd_status",
					Parameters:  map[string]interface{}{"service": serviceName},
				},
				{
					ID:          2,
					Description: fmt.Sprintf("重启 %s 服务", serviceName),
					Tool:        "systemd_restart",
					Parameters:  map[string]interface{}{"service": serviceName},
					Requires:    []int{1},
					Confirm:     true,
				},
			}
		} else {
			plan.Intent = IntentQuery
			plan.Description = "查看服务状态"
			plan.Steps = []Step{
				{
					ID:          1,
					Description: fmt.Sprintf("查看 %s 服务状态", serviceName),
					Tool:        "systemd_status",
					Parameters:  map[string]interface{}{"service": serviceName},
				},
			}
		}
		return plan
	}

	// 日志相关
	if containsAny(input, []string{"日志", "log", "错误", "error", "故障", "问题"}) {
		plan.Intent = IntentTroubleshoot
		plan.Description = "查询日志"
		plan.Steps = []Step{
			{
				ID:          1,
				Description: "查询最近的错误日志",
				Tool:        "log_errors",
				Parameters:  map[string]interface{}{"since": "-1hour", "lines": 20},
			},
		}
		return plan
	}

	// 默认查询
	plan.Intent = IntentQuery
	plan.Description = "信息查询"

	return plan
}

// validatePlan 验证计划
func (p *Planner) validatePlan(plan *Plan) error {
	if plan.Intent == "" {
		return fmt.Errorf("意图不能为空")
	}

	if len(plan.Steps) == 0 {
		return fmt.Errorf("至少需要一个步骤")
	}

	// 验证每个步骤
	for i, step := range plan.Steps {
		if step.ID == 0 {
			return fmt.Errorf("步骤 %d: ID 不能为 0", i+1)
		}

		if step.Tool == "" {
			return fmt.Errorf("步骤 %d: 工具不能为空", i+1)
		}

		// 检查工具是否存在
		if _, ok := p.tools.Get(step.Tool); !ok {
			return fmt.Errorf("步骤 %d: 工具不存在: %s", i+1, step.Tool)
		}
	}

	return nil
}

// EstimateTime 估计执行时间
func (p *Planner) EstimateTime(plan *Plan) int {
	if plan.EstimatedTime > 0 {
		return plan.EstimatedTime
	}

	// 根据步骤估计时间
	totalTime := 0
	for _, step := range plan.Steps {
		switch step.Tool {
		case "monitor_system", "monitor_cpu", "monitor_memory", "monitor_disk":
			totalTime += 2 // 监控工具很快
		case "pacman_search", "pacman_query":
			totalTime += 3
		case "systemd_list", "systemd_status":
			totalTime += 2
		case "log_errors", "log_service":
			totalTime += 5
		case "pacman_install":
			totalTime += 60 // 安装可能需要较长时间
		case "systemd_start", "systemd_stop", "systemd_restart":
			totalTime += 10
		default:
			totalTime += 5
		}
	}

	return totalTime
}

// GetSafetyLevel 获取安全级别
func (p *Planner) GetSafetyLevel(plan *Plan) string {
	if plan.SafetyLevel != "" {
		return plan.SafetyLevel
	}

	// 根据工具判断安全级别
	hasUnsafe := false
	for _, step := range plan.Steps {
		if tool, ok := p.tools.Get(step.Tool); ok && !tool.Safe {
			hasUnsafe = true
			break
		}
	}

	if hasUnsafe {
		return "moderate"
	}
	return "safe"
}

// ExplainPlan 解释计划（返回给用户的说明）
func (p *Planner) ExplainPlan(plan *Plan) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("📋 任务类型: %s\n", translateIntent(plan.Intent)))
	sb.WriteString(fmt.Sprintf("📝 任务描述: %s\n", plan.Description))
	sb.WriteString(fmt.Sprintf("⏱️  预计耗时: %d 秒\n", p.EstimateTime(plan)))
	sb.WriteString(fmt.Sprintf("🛡️  安全级别: %s\n\n", p.GetSafetyLevel(plan)))

	if len(plan.Steps) == 1 {
		sb.WriteString("执行步骤:\n")
		step := plan.Steps[0]
		sb.WriteString(fmt.Sprintf("  1. %s\n", step.Description))
		if step.Confirm {
			sb.WriteString("     ⚠️  此操作需要确认\n")
		}
	} else {
		sb.WriteString(fmt.Sprintf("执行步骤 (共 %d 步):\n", len(plan.Steps)))
		for _, step := range plan.Steps {
			sb.WriteString(fmt.Sprintf("  %d. %s\n", step.ID, step.Description))
			if len(step.Requires) > 0 {
				sb.WriteString(fmt.Sprintf("     依赖步骤: %v\n", step.Requires))
			}
			if step.Confirm {
				sb.WriteString("     ⚠️  此操作需要确认\n")
			}
		}
	}

	return sb.String()
}

// ExecuteStep 执行单个步骤
func (p *Planner) ExecuteStep(ctx context.Context, step Step) (string, error) {
	logger.Info("执行步骤",
		logger.Int("id", step.ID),
		logger.String("tool", step.Tool),
		logger.Any("params", step.Parameters),
	)

	result, err := p.tools.Execute(ctx, step.Tool, step.Parameters)
	if err != nil {
		logger.Error("步骤执行失败",
			logger.Int("id", step.ID),
			logger.Err(err),
		)
		return "", fmt.Errorf("步骤 %d 执行失败: %w", step.ID, err)
	}

	logger.Info("步骤执行成功",
		logger.Int("id", step.ID),
		logger.String("tool", step.Tool),
	)

	return result, nil
}

// translateIntent 翻译意图类型
func translateIntent(intent IntentType) string {
	switch intent {
	case IntentQuery:
		return "查询"
	case IntentPackageManage:
		return "包管理"
	case IntentServiceManage:
		return "服务管理"
	case IntentSystemMonitor:
		return "系统监控"
	case IntentTroubleshoot:
		return "故障排查"
	default:
		return "未知"
	}
}

// containsAny 检查字符串是否包含任意关键词
func containsAny(s string, keywords []string) bool {
	for _, kw := range keywords {
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}

// extractPackageName 从输入中提取包名
func extractPackageName(input string) string {
	// 简单实现：查找最后一个词
	words := strings.Fields(input)
	if len(words) > 0 {
		return words[len(words)-1]
	}
	return "package"
}

// extractServiceName 从输入中提取服务名
func extractServiceName(input string) string {
	// 查找常见服务名
	services := []string{"nginx", "docker", "sshd", "networkmanager", "bluetooth", "cups", "systemd-logind"}
	for _, svc := range services {
		if strings.Contains(input, svc) {
			return svc
		}
	}
	return ""
}

// extractJSON 从文本中提取 JSON
func extractJSON(content string) string {
	// 尝试提取 ```json 代码块
	if strings.Contains(content, "```json") {
		start := strings.Index(content, "```json") + 7
		end := strings.Index(content[start:], "```")
		if end > 0 {
			return strings.TrimSpace(content[start : start+end])
		}
	}

	// 尝试提取 ``` 代码块
	if strings.Contains(content, "```") {
		start := strings.Index(content, "```") + 3
		end := strings.Index(content[start:], "```")
		if end > 0 {
			return strings.TrimSpace(content[start : start+end])
		}
	}

	// 尝试提取 { ... }
	if start := strings.Index(content, "{"); start >= 0 {
		if end := strings.LastIndex(content, "}"); end > start {
			return strings.TrimSpace(content[start : end+1])
		}
	}

	return content
}
