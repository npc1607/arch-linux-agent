package llm

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// SystemPromptBuilder 系统提示词构建器
type SystemPromptBuilder struct {
	safeMode    bool
	toolDesc    string
	systemInfo  string
	rules       string
}

// NewSystemPromptBuilder 创建系统提示词构建器
func NewSystemPromptBuilder() *SystemPromptBuilder {
	return &SystemPromptBuilder{}
}

// SetSafeMode 设置安全模式
func (b *SystemPromptBuilder) SetSafeMode(enabled bool) *SystemPromptBuilder {
	b.safeMode = enabled
	return b
}

// SetToolDesc 设置工具描述
func (b *SystemPromptBuilder) SetToolDesc(desc string) *SystemPromptBuilder {
	b.toolDesc = desc
	return b
}

// SetSystemInfo 设置系统信息
func (b *SystemPromptBuilder) SetSystemInfo(info string) *SystemPromptBuilder {
	b.systemInfo = info
	return b
}

// SetRules 设置规则
func (b *SystemPromptBuilder) SetRules(rules string) *SystemPromptBuilder {
	b.rules = rules
	return b
}

// Build 构建系统提示词
func (b *SystemPromptBuilder) Build() string {
	var prompt strings.Builder

	// 基础描述
	prompt.WriteString("你是一个 Arch Linux 系统管理助手，帮助用户完成系统管理和监控任务。\n\n")

	// 工具描述
	if b.toolDesc != "" {
		prompt.WriteString("可用工具：\n")
		prompt.WriteString(b.toolDesc)
		prompt.WriteString("\n\n")
	}

	// 规则
	prompt.WriteString("规则：\n")
	if b.rules != "" {
		prompt.WriteString(b.rules)
	} else {
		prompt.WriteString(b.getDefaultRules())
	}

	// 安全模式
	if b.safeMode {
		prompt.WriteString("\n\n【安全模式已启用】\n")
		prompt.WriteString("当前处于只读模式，只能执行查询类操作，不能修改系统。\n\n")
		prompt.WriteString("允许的操作：\n")
		prompt.WriteString("- pacman -Ss (搜索包)\n")
		prompt.WriteString("- systemctl status (查看状态)\n")
		prompt.WriteString("- df -h, free -h (查看资源)\n")
		prompt.WriteString("- journalctl (查看日志)\n")
	}

	// 系统信息
	if b.systemInfo != "" {
		prompt.WriteString("\n\n当前系统信息：\n")
		prompt.WriteString(b.systemInfo)
	}

	return prompt.String()
}

// getDefaultRules 获取默认规则
func (b *SystemPromptBuilder) getDefaultRules() string {
	return `1. 只执行用户明确请求的操作
2. 涉及系统变更的操作（安装/删除软件、启停服务）需要告知用户风险
3. 优先使用只读工具获取信息
4. 命令执行失败时分析原因并给出建议
5. 使用简洁专业的中文回复
6. 对于系统状态查询，直接给出结果
7. 对于操作类问题，先说明操作步骤和风险
8. 需要执行命令时，使用明确的命令格式`
}

// GetSystemInfo 获取系统信息
func GetSystemInfo() string {
	var info strings.Builder

	info.WriteString(fmt.Sprintf("操作系统: %s\n", runtime.GOOS))
	info.WriteString(fmt.Sprintf("架构: %s\n", runtime.GOARCH))

	// 内核版本
	if kernelVer := getKernelVersion(); kernelVer != "" {
		info.WriteString(fmt.Sprintf("内核版本: %s\n", kernelVer))
	}

	// 运行时间
	if uptime := getUptime(); uptime != "" {
		info.WriteString(fmt.Sprintf("运行时间: %s\n", uptime))
	}

	// 主机名
	if hostname := getHostname(); hostname != "" {
		info.WriteString(fmt.Sprintf("主机名: %s\n", hostname))
	}

	return info.String()
}

// getKernelVersion 获取内核版本
func getKernelVersion() string {
	cmd := exec.Command("uname", "-r")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// getUptime 获取系统运行时间
func getUptime() string {
	// 读取 /proc/uptime
	data, err := exec.Command("cat", "/proc/uptime").Output()
	if err != nil {
		return ""
	}

	var uptime float64
	_, err = fmt.Sscanf(string(data), "%f", &uptime)
	if err != nil {
		return ""
	}

	// 转换为可读格式
	uptimeSec := int(uptime)
	days := uptimeSec / 86400
	hours := (uptimeSec % 86400) / 3600
	minutes := (uptimeSec % 3600) / 60

	if days > 0 {
		return fmt.Sprintf("%d天 %d小时", days, hours)
	} else if hours > 0 {
		return fmt.Sprintf("%d小时 %d分钟", hours, minutes)
	}
	return fmt.Sprintf("%d分钟", minutes)
}

// getHostname 获取主机名
func getHostname() string {
	cmd := exec.Command("hostname")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// BuildDefaultSystemPrompt 构建默认系统提示词
func BuildDefaultSystemPrompt(safeMode bool) string {
	builder := NewSystemPromptBuilder().
		SetSafeMode(safeMode).
		SetSystemInfo(GetSystemInfo())

	return builder.Build()
}

// BuildSystemPromptWithTools 构建带工具的系统提示词
func BuildSystemPromptWithTools(safeMode bool, toolDesc string) string {
	builder := NewSystemPromptBuilder().
		SetSafeMode(safeMode).
		SetToolDesc(toolDesc).
		SetSystemInfo(GetSystemInfo())

	return builder.Build()
}
