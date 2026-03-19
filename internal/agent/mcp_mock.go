package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/npc1607/arch-linux-agent/internal/config"
	"github.com/npc1607/arch-linux-agent/internal/tools"
	"github.com/npc1607/arch-linux-agent/pkg/logger"
	"github.com/sashabaranov/go-openai"
)

// RealMCPToolsFactory 创建真实 MCP 工具的工厂函数
func RealMCPToolsFactory(serverName string, cfg *config.Config, exec *tools.Executor) ToolLoaderFunc {
	return func(ctx context.Context) ([]*Tool, error) {
		logger.Info("加载真实MCP工具", logger.String("server", serverName))

		switch serverName {
		case "pacman":
			return loadPacmanTools(cfg, exec)
		case "systemd":
			return loadSystemdTools(cfg, exec)
		case "monitor":
			return loadMonitorTools(cfg, exec)
		case "log":
			return loadLogTools(cfg, exec)
		case "weather":
			return loadWeatherTools(cfg)
		default:
			return nil, fmt.Errorf("未知的 MCP 服务器: %s", serverName)
		}
	}
}

// loadPacmanTools 加载 Pacman 包管理工具
func loadPacmanTools(cfg *config.Config, exec *tools.Executor) ([]*Tool, error) {
	pacmanTool := tools.NewPacmanTool(exec, cfg.Commands.PacmanCmd)

	return []*Tool{
		{
			Name:        "pacman_search",
			Description: "搜索软件包",
			Function: openai.FunctionDefinition{
				Name:        "pacman_search",
				Description: "搜索 Arch Linux 软件包",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"keyword": map[string]interface{}{
							"type":        "string",
							"description": "搜索关键词",
						},
					},
					"required": []string{"keyword"},
				},
			},
			Handler: func(ctx context.Context, params map[string]interface{}) (string, error) {
				keyword := params["keyword"].(string)
				packages, err := pacmanTool.Search(ctx, keyword)
				if err != nil {
					return "", err
				}
				return formatPackages(packages), nil
			},
			Safe: true,
		},
		{
			Name:        "pacman_query",
			Description: "查询已安装的软件包信息",
			Function: openai.FunctionDefinition{
				Name:        "pacman_query",
				Description: "查询已安装的软件包详细信息",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"package": map[string]interface{}{
							"type":        "string",
							"description": "软件包名称",
						},
					},
					"required": []string{"package"},
				},
			},
			Handler: func(ctx context.Context, params map[string]interface{}) (string, error) {
				pkgName := params["package"].(string)
				pkg, err := pacmanTool.Query(ctx, pkgName)
				if err != nil {
					return "", err
				}
				return formatPackage(pkg), nil
			},
			Safe: true,
		},
		{
			Name:        "pacman_install",
			Description: "安装软件包（需确认）",
			Function: openai.FunctionDefinition{
				Name:        "pacman_install",
				Description: "安装指定的软件包",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"packages": map[string]interface{}{
							"type":        "array",
							"items":       map[string]interface{}{"type": "string"},
							"description": "要安装的软件包列表",
						},
					},
					"required": []string{"packages"},
				},
			},
			Handler: func(ctx context.Context, params map[string]interface{}) (string, error) {
				packages := params["packages"].([]string)
				result, err := pacmanTool.Install(ctx, packages, false)
				if err != nil {
					return "", err
				}
				return result.Output, nil
			},
			Safe: false,
		},
		{
			Name:        "pacman_check_updates",
			Description: "检查系统更新",
			Function: openai.FunctionDefinition{
				Name:        "pacman_check_updates",
				Description: "检查可用的系统更新",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			},
			Handler: func(ctx context.Context, params map[string]interface{}) (string, error) {
				packages, err := pacmanTool.CheckUpdates(ctx)
				if err != nil {
					return "", err
				}
				return formatPackages(packages), nil
			},
			Safe: true,
		},
	}, nil
}

// loadSystemdTools 加载 Systemd 服务管理工具
func loadSystemdTools(cfg *config.Config, exec *tools.Executor) ([]*Tool, error) {
	systemdTool := tools.NewSystemdTool(exec, "systemctl")

	return []*Tool{
		{
			Name:        "systemd_list",
			Description: "列出系统服务",
			Function: openai.FunctionDefinition{
				Name:        "systemd_list",
				Description: "列出所有系统服务及其状态",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"all": map[string]interface{}{
							"type":        "boolean",
							"description": "是否列出所有服务（包括停止的）",
						},
					},
				},
			},
			Handler: func(ctx context.Context, params map[string]interface{}) (string, error) {
				all := false
				if v, ok := params["all"].(bool); ok {
					all = v
				}
				services, err := systemdTool.ListServices(ctx, all)
				if err != nil {
					return "", err
				}
				return formatServices(services), nil
			},
			Safe: true,
		},
		{
			Name:        "systemd_status",
			Description: "查看服务状态",
			Function: openai.FunctionDefinition{
				Name:        "systemd_status",
				Description: "查看指定服务的详细状态",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"service": map[string]interface{}{
							"type":        "string",
							"description": "服务名称",
						},
					},
					"required": []string{"service"},
				},
			},
			Handler: func(ctx context.Context, params map[string]interface{}) (string, error) {
				service := params["service"].(string)
				svc, err := systemdTool.GetServiceStatus(ctx, service)
				if err != nil {
					return "", err
				}
				return formatService(svc), nil
			},
			Safe: true,
		},
		{
			Name:        "systemd_start",
			Description: "启动服务（需确认）",
			Function: openai.FunctionDefinition{
				Name:        "systemd_start",
				Description: "启动指定的系统服务",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"service": map[string]interface{}{
							"type":        "string",
							"description": "服务名称",
						},
					},
					"required": []string{"service"},
				},
			},
			Handler: func(ctx context.Context, params map[string]interface{}) (string, error) {
				service := params["service"].(string)
				result, err := systemdTool.Start(ctx, service)
				if err != nil {
					return "", err
				}
				return result.Output, nil
			},
			Safe: false,
		},
		{
			Name:        "systemd_stop",
			Description: "停止服务（需确认）",
			Function: openai.FunctionDefinition{
				Name:        "systemd_stop",
				Description: "停止指定的系统服务",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"service": map[string]interface{}{
							"type":        "string",
							"description": "服务名称",
						},
					},
					"required": []string{"service"},
				},
			},
			Handler: func(ctx context.Context, params map[string]interface{}) (string, error) {
				service := params["service"].(string)
				result, err := systemdTool.Stop(ctx, service)
				if err != nil {
					return "", err
				}
				return result.Output, nil
			},
			Safe: false,
		},
		{
			Name:        "systemd_restart",
			Description: "重启服务（需确认）",
			Function: openai.FunctionDefinition{
				Name:        "systemd_restart",
				Description: "重启指定的系统服务",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"service": map[string]interface{}{
							"type":        "string",
							"description": "服务名称",
						},
					},
					"required": []string{"service"},
				},
			},
			Handler: func(ctx context.Context, params map[string]interface{}) (string, error) {
				service := params["service"].(string)
				result, err := systemdTool.Restart(ctx, service)
				if err != nil {
					return "", err
				}
				return result.Output, nil
			},
			Safe: false,
		},
	}, nil
}

// loadMonitorTools 加载系统监控工具
func loadMonitorTools(cfg *config.Config, exec *tools.Executor) ([]*Tool, error) {
	monitorTool := tools.NewMonitorTool(exec)

	return []*Tool{
		{
			Name:        "monitor_cpu",
			Description: "获取 CPU 使用率",
			Function: openai.FunctionDefinition{
				Name:        "monitor_cpu",
				Description: "获取当前 CPU 使用率、核心数、负载等信息",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			},
			Handler: func(ctx context.Context, params map[string]interface{}) (string, error) {
				cpu, err := monitorTool.GetCPUUsage(ctx)
				if err != nil {
					return "", err
				}
				return formatCPUInfo(cpu), nil
			},
			Safe: true,
		},
		{
			Name:        "monitor_memory",
			Description: "获取内存使用情况",
			Function: openai.FunctionDefinition{
				Name:        "monitor_memory",
				Description: "获取内存和交换分区使用情况",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			},
			Handler: func(ctx context.Context, params map[string]interface{}) (string, error) {
				mem, err := monitorTool.GetMemoryUsage(ctx)
				if err != nil {
					return "", err
				}
				return formatMemoryInfo(mem), nil
			},
			Safe: true,
		},
		{
			Name:        "monitor_disk",
			Description: "获取磁盘使用情况",
			Function: openai.FunctionDefinition{
				Name:        "monitor_disk",
				Description: "获取所有挂载点的磁盘使用情况",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			},
			Handler: func(ctx context.Context, params map[string]interface{}) (string, error) {
				disks, err := monitorTool.GetDiskUsage(ctx)
				if err != nil {
					return "", err
				}
				return formatDiskInfo(disks), nil
			},
			Safe: true,
		},
		{
			Name:        "monitor_system",
			Description: "获取系统信息汇总",
			Function: openai.FunctionDefinition{
				Name:        "monitor_system",
				Description: "获取完整的系统信息汇总（CPU、内存、磁盘、运行时间）",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			},
			Handler: func(ctx context.Context, params map[string]interface{}) (string, error) {
				info, err := monitorTool.GetSystemInfo(ctx)
				if err != nil {
					return "", err
				}
				return formatSystemInfo(info), nil
			},
			Safe: true,
		},
	}, nil
}

// loadLogTools 加载日志管理工具
func loadLogTools(cfg *config.Config, exec *tools.Executor) ([]*Tool, error) {
	logTool := tools.NewLogTool(exec)

	return []*Tool{
		{
			Name:        "log_errors",
			Description: "查询系统错误日志",
			Function: openai.FunctionDefinition{
				Name:        "log_errors",
				Description: "查询最近的系统错误日志",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"since": map[string]interface{}{
							"type":        "string",
							"description": "起始时间（如：-1hour, -30min, boot）",
						},
						"lines": map[string]interface{}{
							"type":        "integer",
							"description": "返回行数",
						},
					},
				},
			},
			Handler: func(ctx context.Context, params map[string]interface{}) (string, error) {
				since := ""
				if v, ok := params["since"].(string); ok {
					since = v
				}
				lines := 0
				if v, ok := params["lines"].(int); ok {
					lines = v
				}

				entries, err := logTool.GetSystemErrors(ctx, since, lines)
				if err != nil {
					return "", err
				}
				return formatJournalEntries(entries), nil
			},
			Safe: true,
		},
		{
			Name:        "log_service",
			Description: "查询服务日志",
			Function: openai.FunctionDefinition{
				Name:        "log_service",
				Description: "查询指定服务的日志",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"service": map[string]interface{}{
							"type":        "string",
							"description": "服务名称",
						},
						"lines": map[string]interface{}{
							"type":        "integer",
							"description": "返回行数",
						},
					},
					"required": []string{"service"},
				},
			},
			Handler: func(ctx context.Context, params map[string]interface{}) (string, error) {
				service := params["service"].(string)
				lines := 0
				if v, ok := params["lines"].(float64); ok {
					lines = int(v)
				}

				entries, err := logTool.GetServiceLogs(ctx, service, lines)
				if err != nil {
					return "", err
				}
				return formatJournalEntries(entries), nil
			},
			Safe: true,
		},
	}, nil
}

// loadWeatherTools 加载天气查询工具
func loadWeatherTools(cfg *config.Config) ([]*Tool, error) {
	weatherTool := tools.NewWeatherTool()

	return []*Tool{
		{
			Name:        "weather_get",
			Description: "获取天气信息",
			Function: openai.FunctionDefinition{
				Name:        "weather_get",
				Description: "获取指定地点的当前天气和预报信息。不指定地点则根据IP自动定位",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"location": map[string]interface{}{
							"type":        "string",
							"description": "地点名称（城市名、拼音或英文，如：北京、Shanghai、Tokyo），留空则自动定位",
						},
					},
				},
			},
			Handler: func(ctx context.Context, params map[string]interface{}) (string, error) {
				location := ""
				if v, ok := params["location"].(string); ok {
					location = v
				}

				weather, err := weatherTool.GetWeatherSimple(ctx, location)
				if err != nil {
					return "", err
				}
				return weather, nil
			},
			Safe: true,
		},
	}, nil
}

// SetupRealMCPServers 设置真实的 MCP 服务器工具组
func SetupRealMCPServers(registry *LazyToolRegistry, cfg *config.Config, exec *tools.Executor) {
	// Pacman 包管理工具组
	registry.RegisterLazyGroup("pacman", LazyToolConfig{
		Enabled:        true,
		TriggerKeywords: []string{"包", "软件", "安装", "pacman", "package", "install", "搜索", "卸载"},
		Description:     "Arch Linux 包管理和软件安装工具",
		Category:        "package-management",
	}, RealMCPToolsFactory("pacman", cfg, exec))

	// Systemd 服务管理工具组
	registry.RegisterLazyGroup("systemd", LazyToolConfig{
		Enabled:        true,
		TriggerKeywords: []string{"服务", "systemd", "service", "启动", "停止", "重启", "状态"},
		Description:     "Systemd 服务管理和控制工具",
		Category:        "service-management",
	}, RealMCPToolsFactory("systemd", cfg, exec))

	// 系统监控工具组
	registry.RegisterLazyGroup("monitor", LazyToolConfig{
		Enabled:        true,
		TriggerKeywords: []string{"监控", "CPU", "内存", "磁盘", "系统", "monitor", "system", "资源"},
		Description:     "系统资源监控和性能分析工具",
		Category:        "monitoring",
	}, RealMCPToolsFactory("monitor", cfg, exec))

	// 日志管理工具组
	registry.RegisterLazyGroup("log", LazyToolConfig{
		Enabled:        true,
		TriggerKeywords: []string{"日志", "log", "journal", "错误", "error", "调试"},
		Description:     "系统日志查询和分析工具",
		Category:        "logging",
	}, RealMCPToolsFactory("log", cfg, exec))

	// 天气查询工具组
	registry.RegisterLazyGroup("weather", LazyToolConfig{
		Enabled:        true,
		TriggerKeywords: []string{"天气", "weather", "温度", "气温", "预报"},
		Description:     "天气查询和预报工具",
		Category:        "utilities",
	}, RealMCPToolsFactory("weather", cfg, nil))
}

// FormatLazyToolsStatus 格式化懒加载工具状态
func FormatLazyToolsStatus(status map[string]interface{}) string {
	var sb strings.Builder

	sb.WriteString("# 懒加载工具组状态\n\n")

	for name, info := range status {
		infoMap := info.(map[string]interface{})
		loaded := infoMap["loaded"].(bool)
		description := infoMap["description"].(string)
		category := infoMap["category"].(string)
		toolCount := infoMap["tool_count"].(int)
		keywords := infoMap["keywords"].([]string)

		statusIcon := "❌"
		if loaded {
			statusIcon = "✅"
		}

		sb.WriteString(fmt.Sprintf("## %s %s\n", statusIcon, name))
		sb.WriteString(fmt.Sprintf("**分类**: %s\n", category))
		sb.WriteString(fmt.Sprintf("**描述**: %s\n", description))
		sb.WriteString(fmt.Sprintf("**触发关键词**: %s\n", strings.Join(keywords, ", ")))
		if loaded {
			sb.WriteString(fmt.Sprintf("**工具数量**: %d\n", toolCount))
		} else {
			sb.WriteString("**状态**: 未加载（需要时自动加载）\n")
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
