package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/npc1607/arch-linux-agent/internal/chat"
	"github.com/npc1607/arch-linux-agent/internal/config"
	"github.com/npc1607/arch-linux-agent/internal/llm"
	"github.com/npc1607/arch-linux-agent/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	noStream bool
)

func init() {
	rootCmd.AddCommand(chatCmd)

	chatCmd.Flags().BoolVar(&noStream, "no-stream", false, "禁用流式输出")
}

// chatCmd represents the chat command
var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "启动交互式对话模式",
	Long:  `启动交互式对话模式，可以连续与 AI Agent 进行对话。`,
	Run:   runChat,
}

func runChat(cmd *cobra.Command, args []string) {
	// Override stream flag if --no-stream is set
	if noStream {
		stream = false
	}

	// Load configuration
	cfg, err := loadChatConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	if err := logger.Init(cfg.Logging.ToLoggerConfig()); err != nil {
		fmt.Fprintf(os.Stderr, "初始化日志系统失败: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("启动 Arch Linux AI Agent",
		logger.String("version", "0.1.0"),
		logger.String("mode", "chat"),
		logger.Bool("stream", stream),
		logger.Bool("safe_mode", cfg.Security.SafeMode),
	)

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Info("收到中断信号，正在退出...")
		fmt.Fprintln(os.Stderr, "\n收到中断信号，正在退出...")
		cancel()
	}()

	// Print welcome message
	printWelcome(cfg)

	// Build system prompt
	systemPrompt := llm.BuildDefaultSystemPrompt(cfg.Security.SafeMode)
	logger.Debug("系统提示词已构建",
		logger.Int("length", len(systemPrompt)),
	)

	// Initialize chat session
	session := chat.NewChatSessionWithConfig(ctx, chat.ChatConfig{
		APIKey:       cfg.LLM.APIKey,
		Model:        cfg.LLM.Model,
		BaseURL:      cfg.LLM.BaseURL,
		Stream:       stream,
		Verbose:      verbose,
		SafeMode:     cfg.Security.SafeMode,
		MaxTokens:    cfg.LLM.MaxTokens,
		Temperature:  float64(cfg.LLM.Temperature),
		SystemPrompt: systemPrompt,
	})

	// Read-eval-print loop
	reader := bufio.NewReader(os.Stdin)
	messageCount := 0

	for {
		select {
		case <-ctx.Done():
			logger.Info("退出程序", logger.Int("messages", messageCount))
			return
		default:
			// Print prompt
			fmt.Print("🤖 You> ")

			// Read input
			input, err := reader.ReadString('\n')
			if err != nil {
				logger.Error("读取输入失败", logger.Err(err))
				fmt.Fprintf(os.Stderr, "读取输入错误: %v\n", err)
				continue
			}

			input = strings.TrimSpace(input)

			// Handle empty input
			if input == "" {
				continue
			}

			// Handle exit commands
			if input == "exit" || input == "quit" || input == ":q" {
				logger.Info("用户主动退出", logger.Int("messages", messageCount))
				fmt.Fprintln(os.Stderr, "\n再见！")
				return
			}

			// Handle clear command
			if input == "clear" || input == "cls" {
				session.ClearHistory()
				logger.Debug("对话历史已清空")
				fmt.Fprintln(os.Stderr, "对话历史已清空")
				continue
			}

			// Handle help command
			if input == "help" || input == "?" {
				printChatHelp()
				continue
			}

			// Log user input
			logger.Debug("用户输入",
				logger.String("input", input),
				logger.Int("length", len(input)),
			)

			// Process user input
			startTime := logger.Now()
			if err := session.Process(ctx, input); err != nil {
				logger.Error("处理用户输入失败",
					logger.Err(err),
					logger.String("input", input),
				)
				fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			} else {
				logger.Debug("处理完成",
					logger.String("duration", logger.Now().Sub(startTime).String()),
				)
				messageCount++
			}

			fmt.Println() // Empty line for readability
		}
	}
}

func loadChatConfig() (*config.Config, error) {
	// Determine config file path
	configPath := cfgFile
	if configPath == "" {
		configPath = viper.GetString("config")
	}

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}

	// Override with command line flags
	if apiKey != "" {
		cfg.LLM.APIKey = apiKey
	}
	if model != "" {
		cfg.LLM.Model = model
	}
	if baseURL != "" {
		cfg.LLM.BaseURL = baseURL
	}
	if safeMode {
		cfg.Security.SafeMode = true
	}
	if maxTokens != 0 {
		cfg.LLM.MaxTokens = maxTokens
	}
	if temperature != 0 {
		cfg.LLM.Temperature = temperature
	}

	return cfg, nil
}

func printWelcome(cfg *config.Config) {
	modeStr := "正常模式"
	if cfg.Security.SafeMode {
		modeStr = "安全模式（只读）"
	}

	fmt.Fprintln(os.Stderr, `
╔══════════════════════════════════════════════════════════╗
║          Arch Linux AI Agent v0.1.0                     ║
║          系统管理智能助手                                 ║
╚══════════════════════════════════════════════════════════╝`)

	fmt.Fprintf(os.Stderr, "模式: %s\n", modeStr)
	fmt.Fprintf(os.Stderr, "模型: %s\n", cfg.LLM.Model)
	fmt.Fprintf(os.Stderr, "记忆: %s\n", boolToStr(cfg.Memory.Enabled))
	fmt.Fprintln(os.Stderr, `
输入 "help" 查看可用命令，输入 "exit" 退出
`)
}

func boolToStr(b bool) string {
	if b {
		return "已启用"
	}
	return "已禁用"
}

func printChatHelp() {
	fmt.Fprintln(os.Stderr, `
可用命令:
  help, ?     显示此帮助信息
  clear, cls  清空对话历史
  exit, quit  退出程序

示例对话:
  "检查系统状态"
  "帮我升级系统"
  "查看 CPU 使用率"
  "为什么 nginx 启动失败？"
`)
}
