package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "arch-agent",
	Short: "Arch Linux AI Agent - 系统管理智能助手",
	Long: `Arch Linux AI Agent 是一个基于 AI 的命令行工具，
通过自然语言交互来管理和监控 Arch Linux 系统。

功能包括：
  • 系统管理（包管理、服务管理）
  • 系统监控（资源、日志）
  • 智能助手（命令执行、故障排查）

示例：
  arch-agent              # 启动交互模式
  arch-agent "检查系统"    # 单次执行
  arch-agent chat --no-stream  # 禁用流式输出`,
	Version: "0.1.0",
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Config file
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/arch-agent/config.yaml)")

	// OpenAI API
	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "OpenAI API Key (也可以通过环境变量 OPENAI_API_KEY 设置)")
	rootCmd.PersistentFlags().StringVar(&model, "model", "gpt-4o", "OpenAI 模型 (gpt-4o, gpt-4-turbo, gpt-3.5-turbo)")
	rootCmd.PersistentFlags().StringVar(&baseURL, "base-url", "", "自定义 API Base URL")

	// Behavior
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "详细输出")
	rootCmd.PersistentFlags().BoolVar(&stream, "stream", true, "启用流式输出（默认启用）")
	rootCmd.PersistentFlags().BoolVar(&safeMode, "safe-mode", false, "安全模式（只读，不执行危险操作）")

	// LLM parameters
	rootCmd.PersistentFlags().IntVar(&maxTokens, "max-tokens", 4096, "最大输出 tokens")
	rootCmd.PersistentFlags().Float64Var(&temperature, "temperature", 0.7, "温度参数 (0.0-2.0)")

	// Bind to viper
	viper.BindPFlag("api-key", rootCmd.PersistentFlags().Lookup("api-key"))
	viper.BindPFlag("model", rootCmd.PersistentFlags().Lookup("model"))
	viper.BindPFlag("base-url", rootCmd.PersistentFlags().Lookup("base-url"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("stream", rootCmd.PersistentFlags().Lookup("stream"))
	viper.BindPFlag("safe-mode", rootCmd.PersistentFlags().Lookup("safe-mode"))
	viper.BindPFlag("max-tokens", rootCmd.PersistentFlags().Lookup("max-tokens"))
	viper.BindPFlag("temperature", rootCmd.PersistentFlags().Lookup("temperature"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".arch-agent" (without extension).
		viper.AddConfigPath(filepath.Join(home, ".config", "arch-agent"))
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil && verbose {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}

	// Set API key from environment if not set
	if viper.GetString("api-key") == "" {
		viper.Set("api-key", os.Getenv("OPENAI_API_KEY"))
	}
}
