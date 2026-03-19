package chat

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/npc1607/arch-linux-agent/internal/agent"
	"github.com/npc1607/arch-linux-agent/internal/config"
	"github.com/npc1607/arch-linux-agent/internal/llm"
	"github.com/npc1607/arch-linux-agent/pkg/logger"
	"github.com/npc1607/arch-linux-agent/pkg/thinking"
)

// ChatSession 聊天会话
type ChatSession struct {
	agent        *agent.Agent
	config       ChatConfig
	systemPrompt string
}

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
	SystemPrompt string
}

// NewChatSession 创建新的聊天会话
func NewChatSession(ctx context.Context, cfg ChatConfig) *ChatSession {
	// 创建配置
	config := &config.Config{
		LLM: config.LLMConfig{
			APIKey:      cfg.APIKey,
			Model:       cfg.Model,
			BaseURL:     cfg.BaseURL,
			MaxTokens:   cfg.MaxTokens,
			Temperature: cfg.Temperature,
		},
		Security: config.SecurityConfig{
			SafeMode: cfg.SafeMode,
		},
	}

	// 创建 Agent
	// ag, err := agent.NewAgent(config)
	// 启用懒加载Tools
	ag, err := agent.NewAgentWithLazyLoading(config, true)
	if err != nil {
		logger.Error("创建 Agent 失败", logger.Err(err))
		return &ChatSession{
			config: cfg,
		}
	}

	return &ChatSession{
		agent:        ag,
		config:       cfg,
		systemPrompt: buildSystemPrompt(cfg.SafeMode),
	}
}

// NewChatSessionWithConfig 创建新的聊天会话（使用自定义系统提示词）
func NewChatSessionWithConfig(ctx context.Context, config ChatConfig) *ChatSession {
	return NewChatSession(ctx, config)
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
当前处于只读模式，只执行查询类操作。
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
	// 使用 Agent 处理
	response, err := s.agent.Process(ctx, userInput, nil)
	if err != nil {
		return err
	}

	// 输出响应
	fmt.Printf("🤖 Agent> %s\n", response)

	return nil
}

// ProcessStream 处理用户输入（流式输出）
func (s *ChatSession) ProcessStream(ctx context.Context, userInput string) error {
	// 创建thinking动画（500ms后才显示）
	anim := thinking.NewSpinner("🤖 Agent")

	anim.Start()

	// 用于标记是否已收到第一个chunk
	var hasContent int32

	// 使用 Agent 流式处理
	fullResponse := strings.Builder{}
	err := s.agent.ProcessStream(ctx, userInput, func(chunk string) {
		// 第一个chunk到达时停止动画
		if atomic.CompareAndSwapInt32(&hasContent, 0, 1) {
			anim.Stop()
			// 输出提示
			fmt.Print("🤖 Agent> ")
		}

		// 实时输出每个 token
		fmt.Print(chunk)
		fullResponse.WriteString(chunk)
	})

	// 如果没有收到任何内容，停止动画
	if atomic.LoadInt32(&hasContent) == 0 {
		anim.Stop()
		fmt.Print("🤖 Agent> ")
	}

	if err != nil {
		fmt.Println() // 换行
		return err
	}

	// 确保以换行结束
	if fullResponse.Len() > 0 {
		lastChar := fullResponse.String()[len(fullResponse.String())-1]
		if lastChar != '\n' {
			fmt.Println()
		}
	}

	return nil
}

// ClearHistory 清空对话历史
func (s *ChatSession) ClearHistory() {
	s.agent.ClearHistory()
}

// GetHistory 获取对话历史
func (s *ChatSession) GetHistory() []llm.Message {
	history := s.agent.GetHistory()
	if history == nil {
		return []llm.Message{}
	}
	return history
}

// GetHistoryFormatted 获取格式化的对话历史（用于显示）
func (s *ChatSession) GetHistoryFormatted() []string {
	history := s.agent.GetHistory()
	result := make([]string, len(history))

	for i, msg := range history {
		role := msg.Role
		roleDisplay := role
		switch role {
		case "user":
			roleDisplay = "👤 You"
		case "assistant":
			roleDisplay = "🤖 Agent"
		case "tool":
			roleDisplay = "🔧 Tool"
		}
		result[i] = fmt.Sprintf("%s> %s", roleDisplay, msg.Content)
	}

	return result
}

// UpdateSystemPrompt 更新系统提示
func (s *ChatSession) UpdateSystemPrompt(prompt string) {
	s.systemPrompt = prompt
}
