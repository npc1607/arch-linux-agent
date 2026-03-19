package config

import (
	"github.com/npc1607/arch-linux-agent/pkg/logger"
)

// ToLoggerConfig 转换为 logger 配置
func (c *LoggingConfig) ToLoggerConfig() *logger.Config {
	return &logger.Config{
		Level:      c.Level,
		Output:     c.Output,
		FilePath:   c.FilePath,
		MaxSize:    c.MaxSize,
		MaxBackups: c.MaxBackups,
		MaxAge:     c.MaxAge,
		Compress:   c.Compress,
	}
}
