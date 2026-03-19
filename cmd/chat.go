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
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// chatCmd represents the chat command
var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "启动交互式对话模式",
	Long:  `启动交互式对话模式，可以连续与 AI Agent 进行对话。`,
	Run:   runChat,
}

var (
	noStream bool
)

func init() {
	rootCmd.AddCommand(chatCmd)

	chatCmd.Flags().BoolVar(&noStream, "no-stream", false, "禁用流式输出")
}

func runChat(cmd *cobra.Command, args []string) {
	// Override stream flag if --no-stream is set
	if noStream {
		stream = false
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Fprintln(os.Stderr, "\n收到中断信号，正在退出...")
		cancel()
	}()

	// Print welcome message
	printWelcome()

	// Get API key
	apiKey := viper.GetString("api-key")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "错误: 未设置 API Key")
		fmt.Fprintln(os.Stderr, "请通过以下方式之一设置:")
		fmt.Fprintln(os.Stderr, "  1. 命令行参数: --api-key YOUR_KEY")
		fmt.Fprintln(os.Stderr, "  2. 环境变量: OPENAI_API_KEY=YOUR_KEY")
		fmt.Fprintln(os.Stderr, "  3. 配置文件: ~/.config/arch-agent/config.yaml")
		os.Exit(1)
	}

	// Initialize chat session
	session := chat.NewChatSession(ctx, chat.ChatConfig{
		APIKey:      apiKey,
		Model:       viper.GetString("model"),
		BaseURL:     viper.GetString("base-url"),
		Stream:      stream,
		Verbose:     verbose,
		SafeMode:    safeMode,
		MaxTokens:   maxTokens,
		Temperature: temperature,
	})

	// Read-eval-print loop
	reader := bufio.NewReader(os.Stdin)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Print prompt
			fmt.Print("🤖 You> ")

			// Read input
			input, err := reader.ReadString('\n')
			if err != nil {
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
				fmt.Fprintln(os.Stderr, "\n再见！")
				return
			}

			// Handle clear command
			if input == "clear" || input == "cls" {
				session.ClearHistory()
				fmt.Fprintln(os.Stderr, "对话历史已清空")
				continue
			}

			// Handle help command
			if input == "help" || input == "?" {
				printChatHelp()
				continue
			}

			// Process user input
			if err := session.Process(ctx, input); err != nil {
				fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			}

			fmt.Println() // Empty line for readability
		}
	}
}

func printWelcome() {
	fmt.Fprintln(os.Stderr, `
╔══════════════════════════════════════════════════════════╗
║          Arch Linux AI Agent v0.1.0                     ║
║          系统管理智能助手                                 ║
╚══════════════════════════════════════════════════════════╝

输入 "help" 查看可用命令，输入 "exit" 退出
`)
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
