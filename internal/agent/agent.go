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

	return &Agent{
		config:    cfg,
		llmClient: llmClient,
		tools:     tools,
		messages:  make([]llm.Message, 0),
	}, nil
}

// Process 处理用户输入
func (a *Agent) Process(ctx context.Context, userInput string, streamCallback func(string)) (string, error) {
	logger.Info("Agent 处理用户输入",
		logger.String("input", userInput),
		logger.Int("history_len", len(a.messages)),
	)

	// 添加用户消息
	a.messages = append(a.messages, llm.Message{
		Role:    "user",
		Content: userInput,
	})

	// 构建完整的消息列表（包含系统提示）
	messages := a.buildMessages(userInput)

	// 获取可用工具
	llmTools := a.tools.GetOpenAITools()
	llmFunctions := a.convertToolsToFunctions(llmTools)

	var finalResponse string

	if a.config.LLM.Temperature > 0 && streamCallback != nil {
		// 流式处理
		err := a.llmClient.ChatStream(ctx, a.convertMessagesToLLM(messages), llmFunctions, func(chunk string) {
			streamCallback(chunk)
			finalResponse += chunk
		})

		if err != nil {
			return "", err
		}

		// 保存助手回复
		a.messages = append(a.messages, llm.Message{
			Role:    "assistant",
			Content: finalResponse,
		})

		return finalResponse, nil
	} else {
		// 非流式处理
		response, err := a.llmClient.Chat(ctx, a.convertMessagesToLLM(messages), llmFunctions)
		if err != nil {
			return "", err
		}

		// 处理工具调用
		if len(response.ToolCalls) > 0 {
			return a.handleToolCalls(ctx, response.ToolCalls, streamCallback)
		}

		finalResponse = response.Content

		// 保存助手回复
		a.messages = append(a.messages, llm.Message{
			Role:    "assistant",
			Content: finalResponse,
		})

		return finalResponse, nil
	}
}

// buildMessages 构建消息列表
func (a *Agent) buildMessages(userInput string) []llm.Message {
	// 构建系统提示
	systemPrompt := llm.BuildDefaultSystemPrompt(a.config.Security.SafeMode)

	// 添加工具描述
	tools := a.tools.ListSafe()
	if len(tools) > 0 {
		systemPrompt += "\n\n可用工具：\n"
		for _, tool := range tools {
			systemPrompt += fmt.Sprintf("- %s: %s\n", tool.Name, tool.Description)
		}
	}

	messages := []llm.Message{
		{
			Role:    "system",
			Content: systemPrompt,
		},
	}

	// 添加历史消息
	messages = append(messages, a.messages...)

	return messages
}

// handleToolCalls 处理工具调用
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

	// 让 LLM 总结工具结果
	messages := a.buildMessages("")
	messages = append(messages, a.messages...)

	response, err := a.llmClient.Chat(ctx, a.convertMessagesToLLM(messages), nil)
	if err != nil {
		return "", err
	}

	// 保存最终回复
	a.messages = append(a.messages, llm.Message{
		Role:    "assistant",
		Content: response.Content,
	})

	return response.Content, nil
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
