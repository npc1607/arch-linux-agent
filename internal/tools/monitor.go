package tools

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/npc1607/arch-linux-agent/pkg/logger"
)

// MonitorTool 系统监控工具
type MonitorTool struct {
	exec *Executor
}

// NewMonitorTool 创建监控工具
func NewMonitorTool(exec *Executor) *MonitorTool {
	return &MonitorTool{exec: exec}
}

// CPUInfo CPU 信息
type CPUInfo struct {
	Usage    float64 `json:"usage"`
	Cores    int     `json:"cores"`
	Model    string  `json:"model"`
	LoadAvg  string  `json:"load_avg"`
}

// GetCPUUsage 获取 CPU 使用率
func (m *MonitorTool) GetCPUUsage(ctx context.Context) (*CPUInfo, error) {
	logger.Debug("获取 CPU 使用率")

	// 读取 /proc/stat
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return nil, fmt.Errorf("读取 /proc/stat 失败: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("无效的 /proc/stat 内容")
	}

	// 解析 CPU 时间
	fields := strings.Fields(lines[0])
	if len(fields) < 8 {
		return nil, fmt.Errorf("无效的 CPU 统计数据")
	}

	user, _ := strconv.ParseFloat(fields[1], 64)
	nice, _ := strconv.ParseFloat(fields[2], 64)
	system, _ := strconv.ParseFloat(fields[3], 64)
	idle, _ := strconv.ParseFloat(fields[4], 64)

	total := user + nice + system + idle
	usage := 0.0
	if total > 0 {
		usage = ((total - idle) / total) * 100
	}

	// 获取 CPU 核心数
	cores, _ := os.ReadFile("/sys/devices/system/cpu/present")
	coreCount := strings.Count(string(cores), ",") + 1

	// 获取负载平均值
	loadAvg, _ := os.ReadFile("/proc/loadavg")
	loadParts := strings.Fields(string(loadAvg))
	loadAvgStr := ""
	if len(loadParts) >= 3 {
		loadAvgStr = fmt.Sprintf("%s %s %s", loadParts[0], loadParts[1], loadParts[2])
	}

	return &CPUInfo{
		Usage:   usage,
		Cores:   coreCount,
		LoadAvg: loadAvgStr,
	}, nil
}

// MemoryInfo 内存信息
type MemoryInfo struct {
	Total     uint64  `json:"total"`
	Used      uint64  `json:"used"`
	Free      uint64  `json:"free"`
	Available uint64  `json:"available"`
	Usage     float64 `json:"usage"`
	SwapTotal uint64  `json:"swap_total"`
	SwapUsed  uint64  `json:"swap_used"`
}

// GetMemoryUsage 获取内存使用情况
func (m *MonitorTool) GetMemoryUsage(ctx context.Context) (*MemoryInfo, error) {
	logger.Debug("获取内存使用情况")

	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return nil, fmt.Errorf("读取 /proc/meminfo 失败: %w", err)
	}

	info := &MemoryInfo{}
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		name := strings.TrimSuffix(fields[0], ":")
		value, _ := strconv.ParseUint(fields[1], 10, 64)
		valueKB := value // 值以 KB 为单位

		switch name {
		case "MemTotal":
			info.Total = valueKB
		case "MemFree":
			info.Free = valueKB
		case "MemAvailable":
			info.Available = valueKB
		case "SwapTotal":
			info.SwapTotal = valueKB
		case "SwapFree":
			info.SwapUsed = info.SwapTotal - valueKB
		}
	}

	// 计算已使用内存
	if info.Available > 0 {
		info.Used = info.Total - info.Available
	} else {
		info.Used = info.Total - info.Free
	}

	// 计算使用率
	if info.Total > 0 {
		info.Usage = float64(info.Used) / float64(info.Total) * 100
	}

	return info, nil
}

// DiskInfo 磁盘信息
type DiskInfo struct {
	Filesystem string  `json:"filesystem"`
	Size       uint64  `json:"size"`
	Used       uint64  `json:"used"`
	Available  uint64  `json:"available"`
	Usage      float64 `json:"usage"`
	MountPoint string  `json:"mount_point"`
}

// GetDiskUsage 获取磁盘使用情况
func (m *MonitorTool) GetDiskUsage(ctx context.Context) ([]DiskInfo, error) {
	logger.Debug("获取磁盘使用情况")

	result, err := m.exec.Run(ctx, "df", "-B1", "--output=source,size,usedavail,pcent,target")
	if err != nil {
		return nil, fmt.Errorf("执行 df 命令失败: %w", err)
	}

	disks, err := m.parseDiskUsage(result.Output)
	if err != nil {
		return nil, fmt.Errorf("解析磁盘使用情况失败: %w", err)
	}

	return disks, nil
}

// parseDiskUsage 解析磁盘使用情况
func (m *MonitorTool) parseDiskUsage(output string) ([]DiskInfo, error) {
	var disks []DiskInfo
	lines := strings.Split(output, "\n")

	// 跳过表头
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		disk := DiskInfo{
			Filesystem: fields[0],
			MountPoint: fields[5],
		}

		size, _ := strconv.ParseUint(fields[1], 10, 64)
		used, _ := strconv.ParseUint(fields[2], 10, 64)
		avail, _ := strconv.ParseUint(fields[3], 10, 64)

		disk.Size = size
		disk.Used = used
		disk.Available = avail

		if size > 0 {
			disk.Usage = float64(used) / float64(size) * 100
		}

		disks = append(disks, disk)
	}

	return disks, nil
}

// ProcessInfo 进程信息
type ProcessInfo struct {
	PID     int    `json:"pid"`
	Name    string `json:"name"`
	User    string `json:"user"`
	CPU     float64 `json:"cpu"`
	Memory  float64 `json:"memory"`
	Status  string `json:"status"`
	Command string `json:"command"`
}

// GetProcessList 获取进程列表
func (m *MonitorTool) GetProcessList(ctx context.Context, limit int) ([]ProcessInfo, error) {
	logger.Debug("获取进程列表", logger.Int("limit", limit))

	args := []string{"ps", "aux", "--sort=-%cpu", "--no-headers"}
	if limit > 0 {
		args = append(args, fmt.Sprintf("| head -n %d", limit))
	}

	result, err := m.exec.RunWithTimeout("bash", "-c", strings.Join(args, " "))
	if err != nil {
		return nil, fmt.Errorf("获取进程列表失败: %w", err)
	}

	processes, err := m.parseProcessList(result.Output, limit)
	if err != nil {
		return nil, fmt.Errorf("解析进程列表失败: %w", err)
	}

	return processes, nil
}

// parseProcessList 解析进程列表
func (m *MonitorTool) parseProcessList(output string, limit int) ([]ProcessInfo, error) {
	var processes []ProcessInfo
	lines := strings.Split(output, "\n")

	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 11 {
			continue
		}

		pid, _ := strconv.Atoi(fields[1])
		cpu, _ := strconv.ParseFloat(fields[2], 64)
		mem, _ := strconv.ParseFloat(fields[3], 64)

		process := ProcessInfo{
			PID:     pid,
			User:    fields[0],
			CPU:     cpu,
			Memory:  mem,
			Status:  fields[7],
			Command: strings.Join(fields[10:], " "),
		}

		processes = append(processes, process)
		count++

		if limit > 0 && count >= limit {
			break
		}
	}

	return processes, nil
}

// GetUptime 获取系统运行时间
func (m *MonitorTool) GetUptime(ctx context.Context) (string, error) {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return "", fmt.Errorf("读取 /proc/uptime 失败: %w", err)
	}

	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return "", fmt.Errorf("无效的 /proc/uptime 内容")
	}

	uptime, _ := strconv.ParseFloat(fields[0], 64)

	days := int(uptime / 86400)
	hours := int(uptime/3600) % 24
	minutes := int(uptime/60) % 60

	if days > 0 {
		return fmt.Sprintf("%d天 %d小时 %d分钟", days, hours, minutes), nil
	} else if hours > 0 {
		return fmt.Sprintf("%d小时 %d分钟", hours, minutes), nil
	}
	return fmt.Sprintf("%d分钟", minutes), nil
}

// GetSystemInfo 获取系统信息汇总
func (m *MonitorTool) GetSystemInfo(ctx context.Context) (map[string]interface{}, error) {
	info := make(map[string]interface{})

	// CPU
	cpu, _ := m.GetCPUUsage(ctx)
	info["cpu"] = cpu

	// 内存
	memory, _ := m.GetMemoryUsage(ctx)
	info["memory"] = memory

	// 磁盘
	disks, _ := m.GetDiskUsage(ctx)
	info["disks"] = disks

	// 运行时间
	uptime, _ := m.GetUptime(ctx)
	info["uptime"] = uptime

	// 当前时间
	info["timestamp"] = time.Now().Unix()

	return info, nil
}
