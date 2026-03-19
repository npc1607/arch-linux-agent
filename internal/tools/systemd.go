package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/npc1607/arch-linux-agent/pkg/logger"
)

// SystemdTool systemd 服务管理工具
type SystemdTool struct {
	exec    *Executor
	cmdName string
}

// NewSystemdTool 创建 systemd 工具
func NewSystemdTool(exec *Executor, cmdName string) *SystemdTool {
	if cmdName == "" {
		cmdName = "systemctl"
	}
	return &SystemdTool{
		exec:    exec,
		cmdName: cmdName,
	}
}

// Service 服务信息
type Service struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Loaded      string `json:"loaded"`
	Active      string `json:"active"`
	Sub         string `json:"sub"`
	Enabled     string `json:"enabled"`
}

// ListServices 列出服务
func (s *SystemdTool) ListServices(ctx context.Context, all bool) ([]Service, error) {
	logger.Info("列出服务", logger.Bool("all", all))

	args := []string{"list-units", "--type=service", "--no-pager"}
	if all {
		args = append(args, "--all")
	}

	result, err := s.exec.Run(ctx, s.cmdName, args...)
	if err != nil {
		return nil, fmt.Errorf("列出服务失败: %w", err)
	}

	services := s.parseServiceList(result.Output)
	logger.Info("列出完成", logger.Int("count", len(services)))

	return services, nil
}

// parseServiceList 解析服务列表
func (s *SystemdTool) parseServiceList(output string) []Service {
	var services []Service
	lines := strings.Split(output, "\n")

	// 跳过表头
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 5 {
			services = append(services, Service{
				Name:        strings.TrimSuffix(fields[0], ".service"),
				Loaded:      fields[1],
				Active:      fields[2],
				Sub:         fields[3],
				Description: strings.Join(fields[4:], " "),
			})
		}
	}

	return services
}

// GetServiceStatus 获取服务状态
func (s *SystemdTool) GetServiceStatus(ctx context.Context, name string) (*Service, error) {
	logger.Info("获取服务状态", logger.String("service", name))

	result, err := s.exec.Run(ctx, s.cmdName, "status", name, "--no-pager")
	if err != nil {
		return nil, fmt.Errorf("获取状态失败: %w", err)
	}

	service := s.parseServiceStatus(result.Output)
	service.Name = name

	return service, nil
}

// parseServiceStatus 解析服务状态
func (s *SystemdTool) parseServiceStatus(output string) *Service {
	service := &Service{}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 解析 Loaded 行
		if strings.Contains(line, "Loaded:") {
			if strings.Contains(line, "loaded") {
				service.Loaded = "loaded"
			} else if strings.Contains(line, "not-found") {
				service.Loaded = "not-found"
			} else {
				service.Loaded = "error"
			}

			// 检查是否启用
			if strings.Contains(line, "enabled") {
				service.Enabled = "enabled"
			} else if strings.Contains(line, "disabled") {
				service.Enabled = "disabled"
			} else if strings.Contains(line, "static") {
				service.Enabled = "static"
			}
		}

		// 解析 Active 行
		if strings.Contains(line, "Active:") {
			if strings.Contains(line, "active (running)") {
				service.Active = "active"
				service.Sub = "running"
			} else if strings.Contains(line, "inactive (dead)") {
				service.Active = "inactive"
				service.Sub = "dead"
			} else if strings.Contains(line, "failed") {
				service.Active = "failed"
			}
		}

		// 解析描述
		if strings.Contains(line, "Description:") {
			service.Description = strings.TrimSpace(strings.TrimPrefix(line, "Description:"))
		}
	}

	return service
}

// IsActive 检查服务是否活动
func (s *SystemdTool) IsActive(ctx context.Context, name string) (bool, error) {
	result, err := s.exec.Run(ctx, s.cmdName, "is-active", name)
	if err != nil {
		return false, err
	}

	return strings.TrimSpace(result.Output) == "active", nil
}

// IsEnabled 检查服务是否启用
func (s *SystemdTool) IsEnabled(ctx context.Context, name string) (bool, error) {
	result, err := s.exec.Run(ctx, s.cmdName, "is-enabled", name)
	if err != nil {
		return false, err
	}

	output := strings.TrimSpace(result.Output)
	return output == "enabled" || output == "static", nil
}

// Start 启动服务
func (s *SystemdTool) Start(ctx context.Context, name string) (*CommandResult, error) {
	logger.Info("启动服务", logger.String("service", name))

	result, err := s.exec.Run(ctx, s.cmdName, "start", name)
	if err != nil {
		logger.Error("启动失败", logger.Err(err), logger.String("service", name))
	} else {
		logger.Info("启动成功", logger.String("service", name))
	}

	return result, err
}

// Stop 停止服务
func (s *SystemdTool) Stop(ctx context.Context, name string) (*CommandResult, error) {
	logger.Info("停止服务", logger.String("service", name))

	result, err := s.exec.Run(ctx, s.cmdName, "stop", name)
	if err != nil {
		logger.Error("停止失败", logger.Err(err), logger.String("service", name))
	} else {
		logger.Info("停止成功", logger.String("service", name))
	}

	return result, err
}

// Restart 重启服务
func (s *SystemdTool) Restart(ctx context.Context, name string) (*CommandResult, error) {
	logger.Info("重启服务", logger.String("service", name))

	result, err := s.exec.Run(ctx, s.cmdName, "restart", name)
	if err != nil {
		logger.Error("重启失败", logger.Err(err), logger.String("service", name))
	} else {
		logger.Info("重启成功", logger.String("service", name))
	}

	return result, err
}

// Reload 重载服务配置
func (s *SystemdTool) Reload(ctx context.Context, name string) (*CommandResult, error) {
	logger.Info("重载服务", logger.String("service", name))

	result, err := s.exec.Run(ctx, s.cmdName, "reload", name)
	if err != nil {
		logger.Error("重载失败", logger.Err(err), logger.String("service", name))
	} else {
		logger.Info("重载成功", logger.String("service", name))
	}

	return result, err
}

// Enable 启用服务（开机自启）
func (s *SystemdTool) Enable(ctx context.Context, name string) (*CommandResult, error) {
	logger.Info("启用服务", logger.String("service", name))

	result, err := s.exec.Run(ctx, s.cmdName, "enable", name)
	if err != nil {
		logger.Error("启用失败", logger.Err(err), logger.String("service", name))
	} else {
		logger.Info("启用成功", logger.String("service", name))
	}

	return result, err
}

// Disable 禁用服务
func (s *SystemdTool) Disable(ctx context.Context, name string) (*CommandResult, error) {
	logger.Info("禁用服务", logger.String("service", name))

	result, err := s.exec.Run(ctx, s.cmdName, "disable", name)
	if err != nil {
		logger.Error("禁用失败", logger.Err(err), logger.String("service", name))
	} else {
		logger.Info("禁用成功", logger.String("service", name))
	}

	return result, err
}

// GetJournal 获取服务日志
func (s *SystemdTool) GetJournal(ctx context.Context, name string, lines int) (string, error) {
	logger.Info("获取服务日志",
		logger.String("service", name),
		logger.Int("lines", lines),
	)

	args := []string{"journalctl", "-u", name, "--no-pager"}
	if lines > 0 {
		args = append(args, "-n", fmt.Sprintf("%d", lines))
	}

	result, err := s.exec.Run(ctx, s.cmdName, args...)
	if err != nil {
		return "", fmt.Errorf("获取日志失败: %w", err)
	}

	return result.Output, nil
}

// GetFailedServices 获取失败的服务
func (s *SystemdTool) GetFailedServices(ctx context.Context) ([]Service, error) {
	logger.Info("获取失败的服务")

	result, err := s.exec.Run(ctx, s.cmdName, "list-units", "--failed", "--no-pager")
	if err != nil {
		return nil, fmt.Errorf("获取失败服务失败: %w", err)
	}

	services := s.parseServiceList(result.Output)
	logger.Info("获取完成", logger.Int("count", len(services)))

	return services, nil
}
