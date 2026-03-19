package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/npc1607/arch-linux-agent/pkg/logger"
)

// CommandResult 命令执行结果
type CommandResult struct {
	ExitCode int    `json:"exit_code"`
	Output   string `json:"output"`
	Error    string `json:"error,omitempty"`
	Duration int64  `json:"duration_ms"`
	Success  bool   `json:"success"`
}

// Executor 命令执行器
type Executor struct {
	timeout      time.Duration
	safeMode     bool
	allowSudo    bool
}

// NewExecutor 创建命令执行器
func NewExecutor(timeout time.Duration, safeMode, allowSudo bool) *Executor {
	return &Executor{
		timeout:   timeout,
		safeMode:  safeMode,
		allowSudo: allowSudo,
	}
}

// Run 执行命令
func (e *Executor) Run(ctx context.Context, cmd string, args ...string) (*CommandResult, error) {
	startTime := time.Now()

	logger.Debug("执行命令",
		logger.String("cmd", cmd),
		logger.Any("args", args),
	)

	// 创建命令
	command := exec.CommandContext(ctx, cmd, args...)

	// 捕获输出
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	// 执行命令
	err := command.Run()
	duration := time.Since(startTime)

	result := &CommandResult{
		Output:   strings.TrimSpace(stdout.String()),
		Error:    strings.TrimSpace(stderr.String()),
		Duration: duration.Milliseconds(),
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		}
		result.Success = false
	} else {
		result.ExitCode = 0
		result.Success = true
	}

	logger.Debug("命令执行完成",
		logger.String("cmd", cmd),
		logger.Int("exit_code", result.ExitCode),
		logger.Bool("success", result.Success),
		logger.Int64("duration_ms", result.Duration),
	)

	return result, err
}

// RunWithTimeout 执行命令（带超时）
func (e *Executor) RunWithTimeout(cmd string, args ...string) (*CommandResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

	return e.Run(ctx, cmd, args...)
}

// RunShell 执行 shell 命令（通过 bash -c）
func (e *Executor) RunShell(command string) (*CommandResult, error) {
	return e.RunWithTimeout("bash", "-c", command)
}

// ValidateCommand 验证命令是否安全
func (e *Executor) ValidateCommand(command string) error {
	// 检查命令注入
	if strings.Contains(command, "&&") || strings.Contains(command, "||") {
		return fmt.Errorf("不允许使用命令组合 (&&, ||)")
	}

	if strings.Contains(command, ";") && !strings.Contains(command, "\\;") {
		return fmt.Errorf("不允许使用命令分隔符 (;)")
	}

	// 检查重定向
	if strings.Contains(command, ">") || strings.Contains(command, "<") {
		return fmt.Errorf("不允许使用重定向 (>, <)")
	}

	// 检查管道
	if strings.Contains(command, "|") {
		return fmt.Errorf("不允许使用管道 (|)")
	}

	// 检查后台运行
	if strings.Contains(command, "&") {
		return fmt.Errorf("不允许后台运行 (&)")
	}

	// 安全模式检查
	if e.safeMode {
		// 检查是否包含危险命令
		dangerous := []string{"rm", "mkfs", "dd", "format", "shutdown", "reboot"}
		for _, d := range dangerous {
			if strings.Contains(command, d) {
				return fmt.Errorf("安全模式下不允许执行危险命令: %s", d)
			}
		}
	}

	return nil
}

// IsSafeCommand 检查命令是否安全（只读操作）
func (e *Executor) IsSafeCommand(cmd string) bool {
	safeCommands := []string{
		"pacman -Ss",  // 搜索包
		"pacman -Si",  // 包信息
		"pacman -Q",   // 查询已安装
		"pacman -Ql",  // 查询文件
		"systemctl status",      // 服务状态
		"systemctl is-active",   // 服务活动状态
		"systemctl is-enabled",  // 服务启用状态
		"systemctl list-units",  // 列出服务
		"systemctl list-unit-files",  // 列出服务文件
		"journalctl",  // 日志查询
		"df",          // 磁盘使用
		"df -h",       // 磁盘使用（人类可读）
		"free",        // 内存使用
		"free -h",     // 内存使用（人类可读）
		"uptime",      // 运行时间
		"uname",       // 系统信息
		"cat",         // 查看文件
		"ls",          // 列出文件
		"ps",          // 进程列表
		"hostname",    // 主机名
		"whoami",      // 当前用户
	}

	for _, safe := range safeCommands {
		if strings.HasPrefix(strings.TrimSpace(cmd), safe) {
			return true
		}
	}

	return false
}
