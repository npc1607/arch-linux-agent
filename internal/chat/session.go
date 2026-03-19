package chat

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/sashabaranov/go-openai"
)

// ChatConfig 聊天配置
type ChatConfig struct {
	APIKey       string
	Model        string
	BaseURL      string
	Stream       bool
	Verbose      bool
	SafeMode     bool
	MaxTokens    int
	Temperature  float64
	SystemPrompt string  // 系统提示词
}

// ChatSession 聊天会话
type ChatSession struct {
	client     *openai.Client
	config     ChatConfig
	messages   []openai.ChatCompletionMessage
	systemPrompt string
}

// NewChatSession 创建新的聊天会话
func NewChatSession(ctx context.Context, config ChatConfig) *ChatSession {
	clientConfig := openai.DefaultConfig(config.APIKey)
	if config.BaseURL != "" {
		clientConfig.BaseURL = config.BaseURL
	}

	return &ChatSession{
		client:   openai.NewClientWithConfig(clientConfig),
		config:   config,
		messages: make([]openai.ChatCompletionMessage, 0),
		systemPrompt: buildSystemPrompt(config.SafeMode),
	}
}

// NewChatSessionWithConfig 创建新的聊天会话（使用自定义系统提示词）
func NewChatSessionWithConfig(ctx context.Context, config ChatConfig) *ChatSession {
	clientConfig := openai.DefaultConfig(config.APIKey)
	if config.BaseURL != "" {
		clientConfig.BaseURL = config.BaseURL
	}

	systemPrompt := config.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = buildSystemPrompt(config.SafeMode)
	}

	return &ChatSession{
		client:   openai.NewClientWithConfig(clientConfig),
		config:   config,
		messages: make([]openai.ChatCompletionMessage, 0),
		systemPrompt: systemPrompt,
	}
}

// buildSystemPrompt 构建系统提示词
func buildSystemPrompt(safeMode bool) string {
	prompt := `你是一个 Arch Linux 系统管理助手，帮助用户完成系统管理和监控任务。

可用工具:
- pacman: 包管理 (搜索、安装、删除、升级)
- systemctl: 服务管理 (启动、停止、状态查看)
- 系统监控: CPU、内存、磁盘使用率
- 日志查询: journalctl

规则:
1. 只执行用户明确请求的操作
2. 涉及系统变更的操作需要告知用户风险
3. 优先使用只读工具获取信息
4. 命令执行失败时分析原因并给出建议
5. 使用简洁专业的中文回复

输出格式:
- 对于查询类问题，直接给出结果
- 对于操作类问题，先说明操作步骤和风险
- 需要执行命令时，使用明确的命令格式
`

	if safeMode {
		prompt += `
【安全模式已启用】
当前处于只读模式，不会执行任何修改系统的操作。
只能执行查询类命令，如:
- pacman -Ss (搜索包)
- systemctl status (查看状态)
- df -h, free -h (查看资源)
- journalctl (查看日志)
`
	}

	return prompt
}

// Process 处理用户输入
func (s *ChatSession) Process(ctx context.Context, userInput string) error {
	// 添加用户消息
	s.messages = append(s.messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: userInput,
	})

	// 准备请求消息（包含系统提示）
	allMessages := append([]openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: s.systemPrompt,
		},
	}, s.messages...)

	if s.config.Stream {
		return s.processStream(ctx, allMessages)
	}
	return s.processNormal(ctx, allMessages)
}

// processStream 流式处理
func (s *ChatSession) processStream(ctx context.Context, messages []openai.ChatCompletionMessage) error {
	fmt.Print("🤖 Agent> ")

	request := openai.ChatCompletionRequest{
		Model:       s.config.Model,
		Messages:    messages,
		MaxTokens:   s.config.MaxTokens,
		Temperature: float32(s.config.Temperature),
		Stream:      true,
	}

	stream, err := s.client.CreateChatCompletionStream(ctx, request)
	if err != nil {
		return fmt.Errorf("创建流式请求失败: %w", err)
	}
	defer stream.Close()

	var fullContent strings.Builder

	for {
		response, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("接收流式响应失败: %w", err)
		}

		if len(response.Choices) == 0 {
			continue
		}

		content := response.Choices[0].Delta.Content
		if content != "" {
			fmt.Print(content)
			fullContent.WriteString(content)
		}
	}

	// 保存助手回复
	s.messages = append(s.messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleAssistant,
		Content: fullContent.String(),
	})

	return nil
}

// processNormal 普通处理
func (s *ChatSession) processNormal(ctx context.Context, messages []openai.ChatCompletionMessage) error {
	request := openai.ChatCompletionRequest{
		Model:       s.config.Model,
		Messages:    messages,
		MaxTokens:   s.config.MaxTokens,
		Temperature: float32(s.config.Temperature),
		Stream:      false,
	}

	response, err := s.client.CreateChatCompletion(ctx, request)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	if len(response.Choices) == 0 {
		return fmt.Errorf("未收到响应")
	}

	content := response.Choices[0].Message.Content

	fmt.Printf("🤖 Agent> %s\n", content)

	// 保存助手回复
	s.messages = append(s.messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleAssistant,
		Content: content,
	})

	return nil
}

// ClearHistory 清空对话历史
func (s *ChatSession) ClearHistory() {
	s.messages = make([]openai.ChatCompletionMessage, 0)
}

// GetHistory 获取对话历史
func (s *ChatSession) GetHistory() []openai.ChatCompletionMessage {
	return s.messages
}

// UpdateSystemPrompt 更新系统提示
func (s *ChatSession) UpdateSystemPrompt(prompt string) {
	s.systemPrompt = prompt
}
