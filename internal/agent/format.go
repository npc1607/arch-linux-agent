package agent

import (
	"fmt"
	"strings"

	"github.com/npc1607/arch-linux-agent/internal/tools"
)

// 格式化函数

func formatPackages(packages []tools.Package) string {
	if len(packages) == 0 {
		return "未找到软件包"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("找到 %d 个软件包：\n\n", len(packages)))

	for _, pkg := range packages {
		sb.WriteString(fmt.Sprintf("• %s/%s %s\n", pkg.Repository, pkg.Name, pkg.Version))
		if pkg.Description != "" {
			sb.WriteString(fmt.Sprintf("  %s\n", pkg.Description))
		}
	}

	return sb.String()
}

func formatPackage(pkg *tools.Package) string {
	if pkg == nil {
		return "软件包不存在"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("名称: %s\n", pkg.Name))
	sb.WriteString(fmt.Sprintf("版本: %s\n", pkg.Version))
	if pkg.Repository != "" {
		sb.WriteString(fmt.Sprintf("仓库: %s\n", pkg.Repository))
	}
	if pkg.Description != "" {
		sb.WriteString(fmt.Sprintf("描述: %s\n", pkg.Description))
	}
	if pkg.Size != "" {
		sb.WriteString(fmt.Sprintf("大小: %s\n", pkg.Size))
	}
	sb.WriteString(fmt.Sprintf("已安装: %v\n", pkg.Installed))

	return sb.String()
}

func formatServices(services []tools.Service) string {
	if len(services) == 0 {
		return "未找到服务"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("找到 %d 个服务：\n\n", len(services)))

	sb.WriteString(fmt.Sprintf("%-30s %-10s %-10s %-10s %s\n",
		"服务", "加载", "活动", "子状态", "描述"))
	sb.WriteString(strings.Repeat("-", 80) + "\n")

	for _, svc := range services {
		sb.WriteString(fmt.Sprintf("%-30s %-10s %-10s %-10s %s\n",
			svc.Name, svc.Loaded, svc.Active, svc.Sub, svc.Description))
	}

	return sb.String()
}

func formatService(svc *tools.Service) string {
	if svc == nil {
		return "服务不存在"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("服务名称: %s\n", svc.Name))
	if svc.Description != "" {
		sb.WriteString(fmt.Sprintf("描述: %s\n", svc.Description))
	}
	sb.WriteString(fmt.Sprintf("状态: %s\n", svc.Active))
	sb.WriteString(fmt.Sprintf("子状态: %s\n", svc.Sub))
	sb.WriteString(fmt.Sprintf("加载: %s\n", svc.Loaded))
	if svc.Enabled != "" {
		sb.WriteString(fmt.Sprintf("启用: %s\n", svc.Enabled))
	}

	return sb.String()
}

func formatCPUInfo(cpu *tools.CPUInfo) string {
	if cpu == nil {
		return "无法获取 CPU 信息"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("CPU 使用率: %.1f%%\n", cpu.Usage))
	sb.WriteString(fmt.Sprintf("核心数: %d\n", cpu.Cores))
	if cpu.LoadAvg != "" {
		sb.WriteString(fmt.Sprintf("负载平均: %s\n", cpu.LoadAvg))
	}

	return sb.String()
}

func formatMemoryInfo(mem *tools.MemoryInfo) string {
	if mem == nil {
		return "无法获取内存信息"
	}

	const mb = 1024 * 1024

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("总内存: %.1f GB\n", float64(mem.Total)/mb))
	sb.WriteString(fmt.Sprintf("已使用: %.1f GB (%.1f%%)\n", float64(mem.Used)/mb, mem.Usage))
	sb.WriteString(fmt.Sprintf("可用: %.1f GB\n", float64(mem.Available)/mb))
	if mem.SwapTotal > 0 {
		sb.WriteString(fmt.Sprintf("\n交换分区:\n"))
		sb.WriteString(fmt.Sprintf("总计: %.1f GB\n", float64(mem.SwapTotal)/mb))
		sb.WriteString(fmt.Sprintf("已使用: %.1f GB\n", float64(mem.SwapUsed)/mb))
	}

	return sb.String()
}

func formatDiskInfo(disks []tools.DiskInfo) string {
	if len(disks) == 0 {
		return "无法获取磁盘信息"
	}

	const gb = 1024 * 1024 * 1024

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%-20s %12s %12s %12s %10s %s\n",
		"文件系统", "大小", "已使用", "可用", "使用率", "挂载点"))
	sb.WriteString(strings.Repeat("-", 80) + "\n")

	for _, disk := range disks {
		sb.WriteString(fmt.Sprintf("%-20s %12s %12s %12s %10.1f%% %s\n",
			disk.Filesystem,
			formatBytes(disk.Size, gb),
			formatBytes(disk.Used, gb),
			formatBytes(disk.Available, gb),
			disk.Usage,
			disk.MountPoint))
	}

	return sb.String()
}

func formatBytes(bytes uint64, unit uint64) string {
	if bytes == 0 {
		return "0"
	}
	value := float64(bytes) / float64(unit)
	return fmt.Sprintf("%.1fG", value)
}

func formatSystemInfo(info map[string]interface{}) string {
	var sb strings.Builder

	// CPU
	if cpu, ok := info["cpu"].(*tools.CPUInfo); ok {
		sb.WriteString("\n=== CPU ===\n")
		sb.WriteString(formatCPUInfo(cpu))
	}

	// 内存
	if mem, ok := info["memory"].(*tools.MemoryInfo); ok {
		sb.WriteString("\n=== 内存 ===\n")
		sb.WriteString(formatMemoryInfo(mem))
	}

	// 磁盘
	if disks, ok := info["disks"].([]tools.DiskInfo); ok {
		sb.WriteString("\n=== 磁盘 ===\n")
		sb.WriteString(formatDiskInfo(disks))
	}

	// 运行时间
	if uptime, ok := info["uptime"].(string); ok {
		sb.WriteString(fmt.Sprintf("\n运行时间: %s\n", uptime))
	}

	return sb.String()
}

func formatJournalEntries(entries []tools.JournalEntry) string {
	if len(entries) == 0 {
		return "未找到日志条目"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("找到 %d 条日志：\n\n", len(entries)))

	for _, entry := range entries {
		if entry.Time != "" {
			sb.WriteString(fmt.Sprintf("[%s] ", entry.Time))
		}
		if entry.Process != "" {
			sb.WriteString(fmt.Sprintf("%s: ", entry.Process))
		}
		sb.WriteString(fmt.Sprintf("%s\n", entry.Message))
	}

	return sb.String()
}
