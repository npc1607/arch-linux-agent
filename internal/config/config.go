package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Config 应用配置
type Config struct {
	LLM      LLMConfig      `mapstructure:"llm"`
	Security SecurityConfig `mapstructure:"security"`
	Logging  LoggingConfig  `mapstructure:"logging"`
	Commands CommandsConfig `mapstructure:"commands"`
	Memory   MemoryConfig   `mapstructure:"memory"`
}

// LLMConfig LLM 配置
type LLMConfig struct {
	APIKey  string  `mapstructure:"api_key"`
	Model   string  `mapstructure:"model"`
	BaseURL string  `mapstructure:"base_url"`
	MaxTokens int   `mapstructure:"max_tokens"`
	Temperature float64 `mapstructure:"temperature"`
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	ConfirmBeforeAction bool     `mapstructure:"confirm_before_action"`
	SafeMode            bool     `mapstructure:"safe_mode"`
	MaxExecutionTime    int      `mapstructure:"max_execution_time"` // 秒
	AllowCommands       []string `mapstructure:"allow_commands"`
	DenyCommands        []string `mapstructure:"deny_commands"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level      string `mapstructure:"level"`       // debug, info, warn, error
	Output     string `mapstructure:"output"`      // stdout, file, both
	FilePath   string `mapstructure:"file_path"`   // 日志文件路径
	MaxSize    int    `mapstructure:"max_size"`    // MB
	MaxBackups int    `mapstructure:"max_backups"` // 保留文件数
	MaxAge     int    `mapstructure:"max_age"`     // 天数
	Compress   bool   `mapstructure:"compress"`    // 是否压缩
}

// CommandsConfig 命令配置
type CommandsConfig struct {
	PacmanCmd string `mapstructure:"pacman_cmd"`
	YayCmd    string `mapstructure:"yay_cmd"`
	SudoCmd   string `mapstructure:"sudo_cmd"`
	Systemctl string `mapstructure:"systemctl"`
}

// MemoryConfig 记忆配置
type MemoryConfig struct {
	Enabled          bool     `mapstructure:"enabled"`
	TokenMaxHistory  int      `mapstructure:"token_max_history"`
	LatentMaxMemories int     `mapstructure:"latent_max_memories"`
	AutoSummary      bool     `mapstructure:"auto_summary"`
	DecayInterval    int      `mapstructure:"decay_interval"` // 小时
}

// Load 加载配置
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// 设置默认值
	setDefaults(v)

	// 配置文件搜索路径
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("获取用户目录失败: %w", err)
		}

		v.AddConfigPath(filepath.Join(home, ".config", "arch-agent"))
		v.AddConfigPath(".")
		v.AddConfigPath("/etc/arch-agent")

		v.SetConfigName("config")
		v.SetConfigType("yaml")
	}

	// 环境变量前缀
	v.SetEnvPrefix("ARCH_AGENT")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		// 配置文件不存在不是错误，使用默认值
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("读取配置文件失败: %w", err)
		}
	}

	// 从环境变量覆盖 API Key
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		v.Set("llm.api_key", apiKey)
	}

	// 解析配置
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	return &cfg, nil
}

// setDefaults 设置默认值
func setDefaults(v *viper.Viper) {
	// LLM 默认值
	v.SetDefault("llm.model", "gpt-4o")
	v.SetDefault("llm.max_tokens", 4096)
	v.SetDefault("llm.temperature", 0.7)

	// 安全默认值
	v.SetDefault("security.confirm_before_action", true)
	v.SetDefault("security.safe_mode", false)
	v.SetDefault("security.max_execution_time", 300)

	// 日志默认值
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.output", "stdout")
	v.SetDefault("logging.max_size", 100)
	v.SetDefault("logging.max_backups", 3)
	v.SetDefault("logging.max_age", 7)
	v.SetDefault("logging.compress", true)

	// 命令默认值
	v.SetDefault("commands.pacman_cmd", "pacman")
	v.SetDefault("commands.yay_cmd", "yay")
	v.SetDefault("commands.sudo_cmd", "sudo")
	v.SetDefault("commands.systemctl", "systemctl")

	// 记忆默认值
	v.SetDefault("memory.enabled", true)
	v.SetDefault("memory.token_max_history", 50)
	v.SetDefault("memory.latent_max_memories", 10000)
	v.SetDefault("memory.auto_summary", true)
	v.SetDefault("memory.decay_interval", 24)
}

// Validate 验证配置
func (c *Config) Validate() error {
	// 验证 LLM 配置
	if c.LLM.APIKey == "" {
		return fmt.Errorf("LLM API Key 未设置")
	}

	// 验证模型
	validModels := map[string]bool{
		"gpt-4o":       true,
		"gpt-4-turbo":  true,
		"gpt-4":        true,
		"gpt-3.5-turbo": true,
	}
	if !validModels[c.LLM.Model] {
		return fmt.Errorf("不支持的模型: %s", c.LLM.Model)
	}

	// 验证温度参数
	if c.LLM.Temperature < 0 || c.LLM.Temperature > 2 {
		return fmt.Errorf("temperature 必须在 0-2 之间")
	}

	// 验证日志级别
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLevels[c.Logging.Level] {
		return fmt.Errorf("无效的日志级别: %s", c.Logging.Level)
	}

	// 验证日志输出
	validOutputs := map[string]bool{
		"stdout": true,
		"file":   true,
		"both":   true,
	}
	if !validOutputs[c.Logging.Output] {
		return fmt.Errorf("无效的日志输出: %s", c.Logging.Output)
	}

	// 如果是文件输出，设置默认路径
	if c.Logging.Output == "file" || c.Logging.Output == "both" {
		if c.Logging.FilePath == "" {
			home, err := os.UserHomeDir()
			if err == nil {
				c.Logging.FilePath = filepath.Join(home, ".local", "state", "arch-agent", "agent.log")
			}
		}
		// 确保目录存在
		if err := os.MkdirAll(filepath.Dir(c.Logging.FilePath), 0755); err != nil {
			return fmt.Errorf("创建日志目录失败: %w", err)
		}
	}

	return nil
}

// GetConfigPath 获取配置文件路径
func GetConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "arch-agent", "config.yaml")
}

// GetExampleConfig 获取示例配置
func GetExampleConfig() string {
	return `# Arch Linux AI Agent 配置文件

# OpenAI 配置
llm:
  # API Key (也可以通过环境变量 OPENAI_API_KEY 设置)
  api_key: "sk-your-key-here"

  # 模型选择 (gpt-4o, gpt-4-turbo, gpt-3.5-turbo)
  model: "gpt-4o"

  # 自定义 API 端点 (可选，用于兼容服务)
  base_url: ""

  # 最大输出 tokens
  max_tokens: 4096

  # 温度参数 (0.0-2.0)
  temperature: 0.7

# 安全设置
security:
  # 执行操作前确认
  confirm_before_action: true

  # 安全模式（只读）
  safe_mode: false

  # 最大执行时间（秒）
  max_execution_time: 300

  # 允许的命令白名单（可选）
  allow_commands:
    - "pacman -Ss"
    - "systemctl status"

  # 禁止的命令黑名单（可选）
  deny_commands:
    - "rm -rf"
    - "mkfs"

# 日志配置
logging:
  # 日志级别 (debug, info, warn, error)
  level: "info"

  # 输出方式 (stdout, file, both)
  output: "stdout"

  # 日志文件路径
  file_path: "~/.local/state/arch-agent/agent.log"

  # 单个日志文件最大大小 (MB)
  max_size: 100

  # 保留的旧日志文件数量
  max_backups: 3

  # 保留天数
  max_age: 7

  # 是否压缩旧日志
  compress: true

# 命令配置
commands:
  pacman_cmd: "pacman"
  yay_cmd: "yay"
  sudo_cmd: "sudo"
  systemctl: "systemctl"

# 记忆配置
memory:
  # 是否启用记忆系统
  enabled: true

  # Token 级记忆最大历史数
  token_max_history: 50

  # Latent 级记忆最大数量
  latent_max_memories: 10000

  # 自动摘要
  auto_summary: true

  # 重要性衰减间隔（小时）
  decay_interval: 24
`
}
