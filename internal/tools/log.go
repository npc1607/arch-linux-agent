package tools

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/npc1607/arch-linux-agent/pkg/logger"
)

// LogTool 日志分析工具
type LogTool struct {
	exec *Executor
}

// NewLogTool 创建日志工具
func NewLogTool(exec *Executor) *LogTool {
	return &LogTool{exec: exec}
}

// JournalEntry 日志条目
type JournalEntry struct {
	Time     string `json:"time"`
	Host     string `json:"host"`
	Process  string `json:"process"`
	Message  string `json:"message"`
	Priority int    `json:"priority"`
}

// QueryJournal 查询 systemd 日志
func (l *LogTool) QueryJournal(ctx context.Context, options map[string]interface{}) ([]JournalEntry, error) {
	logger.Info("查询日志", logger.Any("options", options))

	args := []string{"journalctl", "--no-pager"}

	// 添加选项
	if unit, ok := options["unit"].(string); ok && unit != "" {
		args = append(args, "-u", unit)
	}

	if since, ok := options["since"].(string); ok && since != "" {
		args = append(args, "-S", since)
	}

	if until, ok := options["until"].(string); ok && until != "" {
		args = append(args, "-U", until)
	}

	if priority, ok := options["priority"].(int); ok && priority >= 0 {
		args = append(args, "-p", strconv.Itoa(priority))
	}

	if lines, ok := options["lines"].(int); ok && lines > 0 {
		args = append(args, "-n", strconv.Itoa(lines))
	}

	if reverse, ok := options["reverse"].(bool); ok && reverse {
		args = append(args, "--reverse")
	}

	// 使用 output format
	args = append(args, "-o", "json")

	result, err := l.exec.Run(ctx, "journalctl", args...)
	if err != nil {
		return nil, fmt.Errorf("查询日志失败: %w", err)
	}

	entries := l.parseJournalEntries(result.Output)
	logger.Info("查询完成", logger.Int("count", len(entries)))

	return entries, nil
}

// parseJournalEntries 解析日志条目
func (l *LogTool) parseJournalEntries(output string) []JournalEntry {
	var entries []JournalEntry
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// JSON 格式解析（简化版）
		entry := JournalEntry{}

		// 简单解析（实际应该使用 JSON 解析）
		if strings.Contains(line, `"__REALTIME_TIMESTAMP"`) {
			// 提取时间戳
			if idx := strings.Index(line, `"__REALTIME_TIMESTAMP":"`); idx > 0 {
				start := idx + len(`"__REALTIME_TIMESTAMP":"`)
				if end := strings.Index(line[start:], `"`); end > 0 {
					timestamp := line[start : start+end]
					// 转换为可读时间
					if ts, err := strconv.ParseInt(timestamp, 10, 64); err == nil {
						t := time.Unix(ts/1000000, 0)
						entry.Time = t.Format("2006-01-02 15:04:05")
					}
				}
			}
		}

		if strings.Contains(line, `"MESSAGE":"`) {
			// 提取消息
			if idx := strings.Index(line, `"MESSAGE":"`); idx > 0 {
				start := idx + len(`"MESSAGE":"`)
				if end := strings.Index(line[start:], `"`); end > 0 {
					entry.Message = line[start : start+end]
				}
			}
		}

		if strings.Contains(line, `"SYSLOG_IDENTIFIER":"`) {
			// 提取进程名
			if idx := strings.Index(line, `"SYSLOG_IDENTIFIER":"`); idx > 0 {
				start := idx + len(`"SYSLOG_IDENTIFIER":"`)
				if end := strings.Index(line[start:], `"`); end > 0 {
					entry.Process = line[start : start+end]
				}
			}
		}

		if entry.Message != "" {
			entries = append(entries, entry)
		}
	}

	return entries
}

// GetSystemErrors 获取系统错误日志
func (l *LogTool) GetSystemErrors(ctx context.Context, since string, lines int) ([]JournalEntry, error) {
	logger.Info("获取系统错误日志",
		logger.String("since", since),
		logger.Int("lines", lines),
	)

	options := map[string]interface{}{
		"priority": 3, // err 级别
		"reverse": true,
	}

	if since != "" {
		options["since"] = since
	}

	if lines > 0 {
		options["lines"] = lines
	}

	return l.QueryJournal(ctx, options)
}

// GetServiceLogs 获取服务日志
func (l *LogTool) GetServiceLogs(ctx context.Context, unit string, lines int) ([]JournalEntry, error) {
	logger.Info("获取服务日志",
		logger.String("unit", unit),
		logger.Int("lines", lines),
	)

	options := map[string]interface{}{
		"unit":    unit,
		"reverse": true,
	}

	if lines > 0 {
		options["lines"] = lines
	}

	return l.QueryJournal(ctx, options)
}

// GetBootLogs 获取启动日志
func (l *LogTool) GetBootLogs(ctx context.Context) ([]JournalEntry, error) {
	logger.Info("获取启动日志")

	options := map[string]interface{}{
		"since":   "boot",
		"reverse": true,
		"lines":   100,
	}

	return l.QueryJournal(ctx, options)
}

// GetKernelLogs 获取内核日志
func (l *LogTool) GetKernelLogs(ctx context.Context, lines int) (string, error) {
	logger.Info("获取内核日志", logger.Int("lines", lines))

	args := []string{"dmesg", "-T"}
	if lines > 0 {
		args = append(args, fmt.Sprintf("| tail -n %d", lines))
	}

	result, err := l.exec.RunWithTimeout("bash", "-c", strings.Join(args, " "))
	if err != nil {
		return "", fmt.Errorf("获取内核日志失败: %w", err)
	}

	return result.Output, nil
}

// AnalyzeErrors 分析最近的错误
func (l *LogTool) AnalyzeErrors(ctx context.Context) (map[string]int, error) {
	logger.Info("分析错误日志")

	// 获取最近 100 条错误日志
	entries, err := l.GetSystemErrors(ctx, "-1hour", 100)
	if err != nil {
		return nil, err
	}

	// 统计错误类型
	errorCounts := make(map[string]int)

	for _, entry := range entries {
		// 简单的错误分类
		msg := strings.ToLower(entry.Message)

		switch {
		case strings.Contains(msg, "connection refused"):
			errorCounts["connection_refused"]++
		case strings.Contains(msg, "timeout"):
			errorCounts["timeout"]++
		case strings.Contains(msg, "failed to start"):
			errorCounts["service_start_failed"]++
		case strings.Contains(msg, "out of memory"):
			errorCounts["oom"]++
		case strings.Contains(msg, "disk full"):
			errorCounts["disk_full"]++
		case strings.Contains(msg, "permission denied"):
			errorCounts["permission_denied"]++
		default:
			errorCounts["other"]++
		}
	}

	return errorCounts, nil
}
