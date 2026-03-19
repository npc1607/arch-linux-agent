package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/npc1607/arch-linux-agent/internal/chat"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// askCmd represents the ask command
var askCmd = &cobra.Command{
	Use:   "ask [question]",
	Short: "单次提问模式",
	Long:  `单次提问模式，直接回答问题后退出。`,
	Args:  cobra.MinimumNArgs(1),
	Run:   runAsk,
}

func init() {
	rootCmd.AddCommand(askCmd)

	askCmd.Flags().BoolVar(&noStream, "no-stream", false, "禁用流式输出")
}

func runAsk(cmd *cobra.Command, args []string) {
	// Override stream flag if --no-stream is set
	if noStream {
		stream = false
	}

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

	// Combine all arguments into a single question
	question := ""
	for i, arg := range args {
		if i > 0 {
			question += " "
		}
		question += arg
	}

	// Create context
	ctx := context.Background()

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

	// Process user input
	if err := session.Process(ctx, question); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}
