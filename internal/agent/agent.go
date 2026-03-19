package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/npc1607/arch-linux-agent/internal/config"
	"github.com/npc1607/arch-linux-agent/internal/llm"
	"github.com/npc1607/arch-linux-agent/pkg/logger"
	"github.com/sashabaranov/go-openai"
)

// Agent AI Agent
type Agent struct {
	config     *config.Config
	llmClient  *llm.Client
	tools      *ToolRegistry
	planner    *Planner
	messages   []llm.Message
}

// NewAgent 创建 Agent
func NewAgent(cfg *config.Config) (*Agent, error) {
	// 创建 LLM 客户端
	llmClient, err := llm.NewClient(&cfg.LLM)
	if err != nil {
		return nil, fmt.Errorf("创建 LLM 客户端失败: %w", err)
	}

	// 创建工具注册表
	tools := NewToolRegistry(cfg)

	// 创建规划器
	planner := NewPlanner(llmClient, tools)

	return &Agent{
		config:    cfg,
		llmClient: llmClient,
		tools:     tools,
		planner:   planner,
		messages:  make([]llm.Message, 0),
	}, nil
}

// Process 处理用户输入（带规划）
func (a *Agent) Process(ctx context.Context, userInput string, streamCallback func(string)) (string, error) {
	logger.Info("Agent 处理用户输入",
		logger.String("input", userInput),
		logger.Int("history_len", len(a.messages)),
	)

	// 1. 任务规划
	plan, err := a.planner.Plan(ctx, userInput)
	if err != nil {
		return "", fmt.Errorf("任务规划失败: %w", err)
	}

	logger.Info("任务规划完成",
		logger.String("intent", string(plan.Intent)),
		logger.Int("steps", len(plan.Steps)),
		logger.String("safety_level", plan.SafetyLevel),
	)

	// 2. 添加用户消息
	a.messages = append(a.messages, llm.Message{
		Role:    "user",
		Content: userInput,
	})

	// 3. 执行计划
	var results []string
	for _, step := range plan.Steps {
		result, err := a.planner.ExecuteStep(ctx, step)
		if err != nil {
			// 执行失败，记录错误
			errorMsg := fmt.Sprintf("步骤 %d (%s) 执行失败: %v", step.ID, step.Description, err)
			logger.Error(errorMsg)
			results = append(results, errorMsg)
		} else {
			results = append(results, result)
		}
	}

	// 4. 构建响应
	response := a.buildResponse(ctx, userInput, plan, results, streamCallback)

	// 5. 保存助手回复
	a.messages = append(a.messages, llm.Message{
		Role:    "assistant",
		Content: response,
	})

	return response, nil
}

// buildResponse 构建响应
func (a *Agent) buildResponse(ctx context.Context, userInput string, plan *Plan, results []string, streamCallback func(string)) string {
	// 如果只有一个步骤且是查询类，直接返回结果
	if len(plan.Steps) == 1 && plan.Intent == IntentQuery {
		return results[0]
	}

	// 多步骤或复杂任务，让 LLM 总结
	prompt := fmt.Sprintf(`用户请求: %s

执行计划:
- 意图: %s
- 步骤数: %d

执行结果:
%s

请总结执行结果并给出建议。`, userInput, plan.Intent, len(plan.Steps), strings.Join(results, "\n\n"))

	messages := []llm.Message{
		{Role: "system", Content: llm.BuildDefaultSystemPrompt(a.config.Security.SafeMode)},
		{Role: "user", Content: prompt},
	}

	response, err := a.llmClient.Chat(ctx, messages, nil)
	if err != nil {
		// 如果 LLM 总结失败，返回原始结果
		return strings.Join(results, "\n\n")
	}

	return response.Content
}

// handleToolCalls 处理工具调用（保留用于未来可能的直接工具调用）
func (a *Agent) handleToolCalls(ctx context.Context, toolCalls []llm.ToolCall, streamCallback func(string)) (string, error) {
	logger.Info("处理工具调用", logger.Int("count", len(toolCalls)))

	// 添加助手的工具调用消息
	a.messages = append(a.messages, llm.Message{
		Role:    "assistant",
		Content: "",
	})

	var toolResults []string

	for _, tc := range toolCalls {
		logger.Info("执行工具",
			logger.String("tool", tc.Name),
			logger.String("arguments", tc.Arguments),
		)

		// 解析参数
		var params map[string]interface{}
		if err := json.Unmarshal([]byte(tc.Arguments), &params); err != nil {
			logger.Error("解析工具参数失败", logger.Err(err))
			return "", fmt.Errorf("解析工具参数失败: %w", err)
		}

		// 执行工具
		result, err := a.tools.Execute(ctx, tc.Name, params)
		if err != nil {
			logger.Error("工具执行失败",
				logger.String("tool", tc.Name),
				logger.Err(err),
			)

			// 添加错误结果
			a.messages = append(a.messages, llm.Message{
				Role:    "tool",
				Content: fmt.Sprintf("错误: %v", err),
			})

			return "", fmt.Errorf("工具执行失败: %w", err)
		}

		logger.Info("工具执行成功",
			logger.String("tool", tc.Name),
			logger.Int("result_length", len(result)),
		)

		// 添加工具结果
		toolMessage := llm.Message{
			Role:    "tool",
			Content: result,
		}
		a.messages = append(a.messages, toolMessage)

		toolResults = append(toolResults, result)
	}

	// 让 LLM 总结工具结果（使用简化的方式）
	allResults := strings.Join(toolResults, "\n\n")
	return allResults, nil
}

// convertMessagesToLLM 转换消息格式
func (a *Agent) convertMessagesToLLM(messages []llm.Message) []llm.Message {
	return messages
}

// convertToolsToFunctions 转换工具为函数格式
func (a *Agent) convertToolsToFunctions(tools []openai.Tool) []llm.Function {
	functions := make([]llm.Function, 0, len(tools))

	for _, tool := range tools {
		if tool.Function != nil {
			var params map[string]interface{}
			if tool.Function.Parameters != nil {
				// 安全的类型转换
				if p, ok := tool.Function.Parameters.(map[string]interface{}); ok {
					params = p
				} else {
					// 如果不是 map[string]interface{}，创建空 map
					params = make(map[string]interface{})
				}
			}
			functions = append(functions, llm.Function{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				Parameters:  params,
			})
		}
	}

	return functions
}

// ClearHistory 清空对话历史
func (a *Agent) ClearHistory() {
	a.messages = make([]llm.Message, 0)
	logger.Debug("对话历史已清空")
}

// GetHistory 获取对话历史
func (a *Agent) GetHistory() []llm.Message {
	return a.messages
}

// GetToolList 获取工具列表
func (a *Agent) GetToolList() []string {
	tools := a.tools.List()
	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		names = append(names, tool.Name)
	}
	return names
}

// GetToolDescription 获取工具描述
func (a *Agent) GetToolDescription(name string) (string, bool) {
	tool, ok := a.tools.Get(name)
	if !ok {
		return "", false
	}
	return tool.Description, true
}

// IsSafeTool 检查工具是否安全
func (a *Agent) IsSafeTool(name string) bool {
	tool, ok := a.tools.Get(name)
	if !ok {
		return false
	}
	return tool.Safe
}

// ShouldConfirmTool 检查工具是否需要确认
func (a *Agent) ShouldConfirmTool(name string) bool {
	if !a.config.Security.ConfirmBeforeAction {
		return false
	}

	tool, ok := a.tools.Get(name)
	if !ok {
		return true
	}

	// 不安全的工具需要确认
	return !tool.Safe
}

// GetAvailableToolsForLLM 获取 LLM 可用的工具列表
func (a *Agent) GetAvailableToolsForLLM() string {
	tools := a.tools.ListSafe()
	if len(tools) == 0 {
		return "无可用工具"
	}

	var sb strings.Builder
	sb.WriteString("可用工具：\n")

	for _, tool := range tools {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name, tool.Description))
	}

	return sb.String()
}

// UpdateConfig 更新配置
func (a *Agent) UpdateConfig(cfg *config.Config) {
	a.config = cfg
	// 重新创建工具注册表
	a.tools = NewToolRegistry(cfg)
}
