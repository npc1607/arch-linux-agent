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
	config       *config.Config
	llmClient    *llm.Client
	tools        *ToolRegistry
	lazyRegistry *LazyToolRegistry // 懒加载工具注册表（可选）
	planner      *Planner
	messages     []llm.Message
	usePlanner   bool // 是否使用规划模式
}

// NewAgent 创建 Agent（传统方式，预加载所有工具）
func NewAgent(cfg *config.Config) (*Agent, error) {
	// 创建 LLM 客户端
	llmClient, err := llm.NewClient(&cfg.LLM)
	if err != nil {
		return nil, fmt.Errorf("创建 LLM 客户端失败: %w", err)
	}

	// 创建工具注册表（预加载所有工具）
	tools := NewToolRegistry(cfg)

	// 创建规划器
	planner := NewPlanner(llmClient, tools)

	return &Agent{
		config:     cfg,
		llmClient:  llmClient,
		tools:      tools,
		planner:    planner,
		messages:   make([]llm.Message, 0),
		usePlanner: false, // 默认使用 Function Calling
	}, nil
}

// NewAgentWithLazyLoading 创建支持懒加载的 Agent
// 懒加载模式下，工具不会在启动时全部加载，而是根据用户消息按需加载
// 这样可以大幅减少启动时间和 token 消耗
func NewAgentWithLazyLoading(cfg *config.Config, enableLazy bool) (*Agent, error) {
	// 创建 LLM 客户端
	llmClient, err := llm.NewClient(&cfg.LLM)
	if err != nil {
		return nil, fmt.Errorf("创建 LLM 客户端失败: %w", err)
	}

	// 创建基础工具注册表（只包含核心工具）
	tools := NewToolRegistry(cfg)

	var lazyRegistry *LazyToolRegistry
	if enableLazy {
		// 创建懒加载注册表
		lazyRegistry = NewLazyToolRegistry(tools)

		// 获取命令执行器（从基础工具注册表中获取）
		exec := tools.GetExecutor()

		// 设置懒加载工具组
		SetupRealMCPServers(lazyRegistry, cfg, exec)

		logger.Info("懒加载工具注册表已创建",
			logger.Bool("enabled", true),
		)
	}

	// 创建规划器
	planner := NewPlanner(llmClient, tools)

	return &Agent{
		config:       cfg,
		llmClient:    llmClient,
		tools:        tools,
		lazyRegistry: lazyRegistry,
		planner:      planner,
		messages:     make([]llm.Message, 0),
		usePlanner:   false,
	}, nil
}

// Process 处理用户输入（支持 Function Calling 和规划模式）
func (a *Agent) Process(ctx context.Context, userInput string, streamCallback func(string)) (string, error) {
	logger.Info("Agent 处理用户输入",
		logger.String("input", userInput),
		logger.Bool("use_planner", a.usePlanner),
		logger.Int("history_len", len(a.messages)),
	)

	// 检查是否需要加载懒加载工具
	if a.lazyRegistry != nil {
		logger.Info("检查懒加载工具", logger.String("input", userInput))
		loadedGroups, err := a.lazyRegistry.CheckAndLoad(ctx, userInput)
		if IsToolLoadPending(err) {
			// 需要用户确认
			groupName, _ := GetGroupNameFromError(err)
			logger.Info("检测到需要加载工具组，自动确认",
				logger.String("group", groupName),
			)

			// 自动确认并加载
			_, err = a.lazyRegistry.LoadGroup(ctx, groupName, true)
			if err != nil {
				logger.Error("加载工具组失败", logger.Err(err))
				return "", fmt.Errorf("加载工具组 %s 失败: %w", groupName, err)
			}
			logger.Info("工具组加载成功", logger.Strings("groups", loadedGroups))
		} else if err != nil {
			logger.Error("检查工具加载失败", logger.Err(err))
		} else if len(loadedGroups) > 0 {
			logger.Info("自动加载了工具组", logger.Strings("groups", loadedGroups))
		} else {
			logger.Info("无需加载新工具组")
		}
	}

	// 添加用户消息
	a.messages = append(a.messages, llm.Message{
		Role:    "user",
		Content: userInput,
	})

	if a.usePlanner {
		return a.processWithPlanner(ctx, userInput, streamCallback)
	}

	return a.processWithFunctionCalling(ctx, userInput, streamCallback)
}

// ProcessStream 处理用户输入（流式输出）
func (a *Agent) ProcessStream(ctx context.Context, userInput string, streamCallback func(string)) error {
	logger.Info("Agent 流式处理用户输入",
		logger.String("input", userInput),
		logger.Bool("use_planner", a.usePlanner),
	)

	// 检查是否需要加载懒加载工具
	if a.lazyRegistry != nil {
		loadedGroups, err := a.lazyRegistry.CheckAndLoad(ctx, userInput)
		if IsToolLoadPending(err) {
			// 需要用户确认
			groupName, _ := GetGroupNameFromError(err)
			logger.Info("检测到需要加载工具组，自动确认",
				logger.String("group", groupName),
			)

			// 自动确认并加载
			_, err = a.lazyRegistry.LoadGroup(ctx, groupName, true)
			if err != nil {
				logger.Error("加载工具组失败", logger.Err(err))
				return fmt.Errorf("加载工具组 %s 失败: %w", groupName, err)
			}
			logger.Info("工具组加载成功", logger.Strings("groups", loadedGroups))
		} else if err != nil {
			logger.Error("检查工具加载失败", logger.Err(err))
		} else if len(loadedGroups) > 0 {
			logger.Info("自动加载了工具组", logger.Strings("groups", loadedGroups))
		}
	}

	// 添加用户消息
	a.messages = append(a.messages, llm.Message{
		Role:    "user",
		Content: userInput,
	})

	if a.usePlanner {
		_, err := a.processWithPlanner(ctx, userInput, streamCallback)
		return err
	}

	// 构建消息列表
	messages := a.buildMessages()

	// 获取可用工具（转换为 OpenAI 格式）
	var llmTools []openai.Tool
	if a.lazyRegistry != nil {
		// 使用懒加载注册表获取工具（包含已加载的懒加载工具）
		llmTools = a.lazyRegistry.GetLoadedTools()
		logger.Info("使用懒加载注册表获取工具", logger.Int("count", len(llmTools)))
	} else {
		// 使用传统方式获取工具
		llmTools = a.tools.GetOpenAITools()
		logger.Info("使用传统方式获取工具", logger.Int("count", len(llmTools)))
	}

	llmFunctions := a.convertToolsToFunctions(llmTools)

	// 第一步：先用非流式检测是否需要工具调用
	response, err := a.llmClient.Chat(ctx, a.convertMessagesToLLM(messages), llmFunctions)
	if err != nil {
		return err
	}

	// 如果有工具调用，使用 Function Calling 模式处理
	if len(response.ToolCalls) > 0 {
		logger.Info("检测到工具调用，切换到 Function Calling 模式")
		finalResponse, err := a.handleToolCalls(ctx, response.ToolCalls, streamCallback)
		if err != nil {
			return err
		}

		// 流式输出最终响应
		if streamCallback != nil {
			for _, ch := range finalResponse {
				streamCallback(string(ch))
			}
		}
		return nil
	}

	// 如果不需要工具，使用流式输出回复
	logger.Info("无需工具调用，使用流式输出")
	if response.Content != "" {
		// 有初始响应，流式输出
		if streamCallback != nil {
			for _, ch := range response.Content {
				streamCallback(string(ch))
			}
		}

		// 保存助手回复
		a.messages = append(a.messages, llm.Message{
			Role:    "assistant",
			Content: response.Content,
		})
		return nil
	}

	// 如果 LLM 没有返回任何内容，调用流式接口
	var fullContent strings.Builder
	err = a.llmClient.ChatStream(ctx, a.convertMessagesToLLM(messages), nil, func(chunk string) {
		fullContent.WriteString(chunk)
		if streamCallback != nil {
			streamCallback(chunk)
		}
	})

	if err != nil {
		return err
	}

	// 保存助手回复
	a.messages = append(a.messages, llm.Message{
		Role:    "assistant",
		Content: fullContent.String(),
	})

	return nil
}

// processWithFunctionCalling 使用 Function Calling 模式
func (a *Agent) processWithFunctionCalling(ctx context.Context, userInput string, streamCallback func(string)) (string, error) {
	logger.Info("使用 Function Calling 模式")

	// 构建消息列表
	messages := a.buildMessages()

	// 获取可用工具（转换为 OpenAI 格式）
	var llmTools []openai.Tool
	if a.lazyRegistry != nil {
		// 使用懒加载注册表获取工具（包含已加载的懒加载工具）
		llmTools = a.lazyRegistry.GetLoadedTools()
		logger.Info("使用懒加载注册表获取工具", logger.Int("count", len(llmTools)))
	} else {
		// 使用传统方式获取工具
		llmTools = a.tools.GetOpenAITools()
		logger.Info("使用传统方式获取工具", logger.Int("count", len(llmTools)))
	}

	llmFunctions := a.convertToolsToFunctions(llmTools)

	logger.Info("可用工具列表",
		logger.Int("total", len(llmFunctions)),
	)

	// 打印工具名称用于调试
	for i, fn := range llmFunctions {
		if i < 10 { // 只打印前10个
			logger.Debug("工具", logger.String("name", fn.Name))
		}
	}

	// 调用 LLM
	response, err := a.llmClient.Chat(ctx, a.convertMessagesToLLM(messages), llmFunctions)
	if err != nil {
		return "", err
	}

	logger.Info("LLM 响应",
		logger.Bool("has_content", response.Content != ""),
		logger.Int("tool_calls", len(response.ToolCalls)),
	)

	// 如果有工具调用，执行工具
	if len(response.ToolCalls) > 0 {
		return a.handleToolCalls(ctx, response.ToolCalls, streamCallback)
	}

	// 没有工具调用，直接返回 LLM 响应
	finalResponse := response.Content

	// 保存助手回复
	a.messages = append(a.messages, llm.Message{
		Role:    "assistant",
		Content: finalResponse,
	})

	return finalResponse, nil
}

// processWithPlanner 使用规划模式
func (a *Agent) processWithPlanner(ctx context.Context, userInput string, streamCallback func(string)) (string, error) {
	logger.Info("使用规划模式")

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

	// 2. 执行计划
	var results []string
	for _, step := range plan.Steps {
		// 检查是否需要确认
		if step.Confirm && a.config.Security.ConfirmBeforeAction {
			// TODO: 添加用户确认逻辑
			logger.Info("需要用户确认", logger.Int("step", step.ID))
		}

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

	// 3. 构建响应
	response := a.buildResponseFromPlan(ctx, userInput, plan, results)

	// 4. 保存助手回复
	a.messages = append(a.messages, llm.Message{
		Role:    "assistant",
		Content: response,
	})

	return response, nil
}

// buildMessages 构建消息列表（包含系统提示和历史）
func (a *Agent) buildMessages() []llm.Message {
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

// handleToolCalls 处理工具调用（Function Calling）
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
	return a.summarizeToolResults(ctx, toolResults, streamCallback)
}

// summarizeToolResults 让 LLM 总结工具执行结果
func (a *Agent) summarizeToolResults(ctx context.Context, toolResults []string, streamCallback func(string)) (string, error) {
	prompt := fmt.Sprintf(`工具执行结果：
%s

请总结以上结果并给出回复。`, strings.Join(toolResults, "\n\n"))

	messages := a.buildMessages()
	messages = append(messages, llm.Message{
		Role:    "user",
		Content: prompt,
	})

	// 如果提供了流式回调，使用流式输出
	if streamCallback != nil {
		var fullContent strings.Builder
		err := a.llmClient.ChatStream(ctx, a.convertMessagesToLLM(messages), nil, func(chunk string) {
			fullContent.WriteString(chunk)
			streamCallback(chunk)
		})

		if err != nil {
			// 如果 LLM 总结失败，返回原始结果
			result := strings.Join(toolResults, "\n\n")
			for _, ch := range result {
				streamCallback(string(ch))
			}
			return result, nil
		}

		// 保存最终回复
		a.messages = append(a.messages, llm.Message{
			Role:    "assistant",
			Content: fullContent.String(),
		})

		return fullContent.String(), nil
	}

	// 非流式模式
	response, err := a.llmClient.Chat(ctx, a.convertMessagesToLLM(messages), nil)
	if err != nil {
		// 如果 LLM 总结失败，返回原始结果
		return strings.Join(toolResults, "\n\n"), nil
	}

	// 保存最终回复
	a.messages = append(a.messages, llm.Message{
		Role:    "assistant",
		Content: response.Content,
	})

	return response.Content, nil
}

// buildResponseFromPlan 从计划结果构建响应
func (a *Agent) buildResponseFromPlan(ctx context.Context, userInput string, plan *Plan, results []string) string {
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

// SetPlannerMode 设置规划模式
func (a *Agent) SetPlannerMode(enable bool) {
	a.usePlanner = enable
	logger.Info("规划模式", logger.Bool("enabled", enable))
}

// UpdateConfig 更新配置
func (a *Agent) UpdateConfig(cfg *config.Config) {
	a.config = cfg
	// 重新创建工具注册表
	a.tools = NewToolRegistry(cfg)
}
