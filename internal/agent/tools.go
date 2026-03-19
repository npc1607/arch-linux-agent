package agent

import (
	"context"
	"fmt"

	"github.com/npc1607/arch-linux-agent/internal/config"
	"github.com/npc1607/arch-linux-agent/internal/tools"
	"github.com/npc1607/arch-linux-agent/pkg/logger"
	"github.com/sashabaranov/go-openai"
)

// Tool 工具定义
type Tool struct {
	Name        string                     `json:"name"`
	Description string                     `json:"description"`
	Function    openai.FunctionDefinition  `json:"function"`
	Handler     ToolHandler                `json:"-"`
	Safe        bool                       `json:"safe"`
}

// ToolHandler 工具处理函数
type ToolHandler func(ctx context.Context, params map[string]interface{}) (string, error)

// ToolRegistry 工具注册表
type ToolRegistry struct {
	tools   map[string]*Tool
	config  *config.Config
	exec    *tools.Executor
	pacman  *tools.PacmanTool
	systemd *tools.SystemdTool
	monitor *tools.MonitorTool
	log     *tools.LogTool
	weather *tools.WeatherTool
}

// NewToolRegistry 创建工具注册表
func NewToolRegistry(cfg *config.Config) *ToolRegistry {
	exec := tools.NewExecutor(
		300*1000000000, // 5分钟
		cfg.Security.SafeMode,
		true, // 允许 sudo
	)

	registry := &ToolRegistry{
		tools:  make(map[string]*Tool),
		config: cfg,
		exec:   exec,
	}

	// 初始化工具
	registry.pacman = tools.NewPacmanTool(exec, cfg.Commands.PacmanCmd)
	registry.systemd = tools.NewSystemdTool(exec, "systemctl")
	registry.monitor = tools.NewMonitorTool(exec)
	registry.log = tools.NewLogTool(exec)
	registry.weather = tools.NewWeatherTool()

	// 注册所有工具
	registry.registerAll()

	return registry
}

// registerAll 注册所有工具
func (r *ToolRegistry) registerAll() {
	// 包管理工具
	r.Register(&Tool{
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
		Handler: r.pacmanSearch,
		Safe:    true,
	})

	r.Register(&Tool{
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
		Handler: r.pacmanQuery,
		Safe:    true,
	})

	r.Register(&Tool{
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
		Handler: r.pacmanInstall,
		Safe:    false,
	})

	r.Register(&Tool{
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
		Handler: r.pacmanCheckUpdates,
		Safe:    true,
	})

	// 服务管理工具
	r.Register(&Tool{
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
		Handler: r.systemdList,
		Safe:    true,
	})

	r.Register(&Tool{
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
		Handler: r.systemdStatus,
		Safe:    true,
	})

	r.Register(&Tool{
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
		Handler: r.systemdStart,
		Safe:    false,
	})

	r.Register(&Tool{
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
		Handler: r.systemdStop,
		Safe:    false,
	})

	r.Register(&Tool{
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
		Handler: r.systemdRestart,
		Safe:    false,
	})

	// 系统监控工具
	r.Register(&Tool{
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
		Handler: r.monitorCPU,
		Safe:    true,
	})

	r.Register(&Tool{
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
		Handler: r.monitorMemory,
		Safe:    true,
	})

	r.Register(&Tool{
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
		Handler: r.monitorDisk,
		Safe:    true,
	})

	r.Register(&Tool{
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
		Handler: r.monitorSystem,
		Safe:    true,
	})

	// 日志工具
	r.Register(&Tool{
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
		Handler: r.logErrors,
		Safe:    true,
	})

	r.Register(&Tool{
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
		Handler: r.logService,
		Safe:    true,
	})

	// 天气工具
	r.Register(&Tool{
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
		Handler: r.weatherGet,
		Safe:    true,
	})

	logger.Info("工具注册完成", logger.Int("count", len(r.tools)))
}

// Register 注册工具
func (r *ToolRegistry) Register(tool *Tool) {
	r.tools[tool.Name] = tool
}

// Get 获取工具
func (r *ToolRegistry) Get(name string) (*Tool, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

// GetExecutor 获取命令执行器（供懒加载使用）
func (r *ToolRegistry) GetExecutor() *tools.Executor {
	return r.exec
}

// List 列出所有工具
func (r *ToolRegistry) List() []*Tool {
	tools := make([]*Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// ListSafe 列出安全工具（只读）
func (r *ToolRegistry) ListSafe() []*Tool {
	tools := make([]*Tool, 0)
	for _, tool := range r.tools {
		if tool.Safe {
			tools = append(tools, tool)
		}
	}
	return tools
}

// GetOpenAITools 转换为 OpenAI 工具格式
func (r *ToolRegistry) GetOpenAITools() []openai.Tool {
	tools := make([]openai.Tool, 0, len(r.tools))

	for _, tool := range r.tools {
		// 如果是安全模式，只返回安全工具
		if r.config.Security.SafeMode && !tool.Safe {
			continue
		}

		tools = append(tools, openai.Tool{
			Type:     "function",
			Function: &tool.Function,
		})
	}

	return tools
}

// Execute 执行工具
func (r *ToolRegistry) Execute(ctx context.Context, name string, params map[string]interface{}) (string, error) {
	tool, ok := r.Get(name)
	if !ok {
		return "", fmt.Errorf("工具不存在: %s", name)
	}

	// 安全模式检查
	if r.config.Security.SafeMode && !tool.Safe {
		return "", fmt.Errorf("安全模式下不允许执行此工具: %s", name)
	}

	logger.Info("执行工具",
		logger.String("tool", name),
		logger.Bool("safe", tool.Safe),
		logger.Any("params", params),
	)

	result, err := tool.Handler(ctx, params)
	if err != nil {
		logger.Error("工具执行失败",
			logger.String("tool", name),
			logger.Err(err),
		)
		return "", fmt.Errorf("工具 %s 执行失败: %w", name, err)
	}

	logger.Info("工具执行成功",
		logger.String("tool", name),
		logger.Int("result_length", len(result)),
	)

	return result, nil
}

// 以下是工具处理函数

func (r *ToolRegistry) pacmanSearch(ctx context.Context, params map[string]interface{}) (string, error) {
	keyword := params["keyword"].(string)
	packages, err := r.pacman.Search(ctx, keyword)
	if err != nil {
		return "", err
	}
	return formatPackages(packages), nil
}

func (r *ToolRegistry) pacmanQuery(ctx context.Context, params map[string]interface{}) (string, error) {
	pkgName := params["package"].(string)
	pkg, err := r.pacman.Query(ctx, pkgName)
	if err != nil {
		return "", err
	}
	return formatPackage(pkg), nil
}

func (r *ToolRegistry) pacmanInstall(ctx context.Context, params map[string]interface{}) (string, error) {
	packages := params["packages"].([]string)
	result, err := r.pacman.Install(ctx, packages, false)
	if err != nil {
		return "", err
	}
	return result.Output, nil
}

func (r *ToolRegistry) pacmanCheckUpdates(ctx context.Context, params map[string]interface{}) (string, error) {
	packages, err := r.pacman.CheckUpdates(ctx)
	if err != nil {
		return "", err
	}
	return formatPackages(packages), nil
}

func (r *ToolRegistry) systemdList(ctx context.Context, params map[string]interface{}) (string, error) {
	all := false
	if v, ok := params["all"].(bool); ok {
		all = v
	}
	services, err := r.systemd.ListServices(ctx, all)
	if err != nil {
		return "", err
	}
	return formatServices(services), nil
}

func (r *ToolRegistry) systemdStatus(ctx context.Context, params map[string]interface{}) (string, error) {
	service := params["service"].(string)
	svc, err := r.systemd.GetServiceStatus(ctx, service)
	if err != nil {
		return "", err
	}
	return formatService(svc), nil
}

func (r *ToolRegistry) systemdStart(ctx context.Context, params map[string]interface{}) (string, error) {
	service := params["service"].(string)
	result, err := r.systemd.Start(ctx, service)
	if err != nil {
		return "", err
	}
	return result.Output, nil
}

func (r *ToolRegistry) systemdStop(ctx context.Context, params map[string]interface{}) (string, error) {
	service := params["service"].(string)
	result, err := r.systemd.Stop(ctx, service)
	if err != nil {
		return "", err
	}
	return result.Output, nil
}

func (r *ToolRegistry) systemdRestart(ctx context.Context, params map[string]interface{}) (string, error) {
	service := params["service"].(string)
	result, err := r.systemd.Restart(ctx, service)
	if err != nil {
		return "", err
	}
	return result.Output, nil
}

func (r *ToolRegistry) monitorCPU(ctx context.Context, params map[string]interface{}) (string, error) {
	cpu, err := r.monitor.GetCPUUsage(ctx)
	if err != nil {
		return "", err
	}
	return formatCPUInfo(cpu), nil
}

func (r *ToolRegistry) monitorMemory(ctx context.Context, params map[string]interface{}) (string, error) {
	mem, err := r.monitor.GetMemoryUsage(ctx)
	if err != nil {
		return "", err
	}
	return formatMemoryInfo(mem), nil
}

func (r *ToolRegistry) monitorDisk(ctx context.Context, params map[string]interface{}) (string, error) {
	disks, err := r.monitor.GetDiskUsage(ctx)
	if err != nil {
		return "", err
	}
	return formatDiskInfo(disks), nil
}

func (r *ToolRegistry) monitorSystem(ctx context.Context, params map[string]interface{}) (string, error) {
	info, err := r.monitor.GetSystemInfo(ctx)
	if err != nil {
		return "", err
	}
	return formatSystemInfo(info), nil
}

func (r *ToolRegistry) logErrors(ctx context.Context, params map[string]interface{}) (string, error) {
	since := ""
	if v, ok := params["since"].(string); ok {
		since = v
	}
	lines := 0
	if v, ok := params["lines"].(int); ok {
		lines = v
	}

	entries, err := r.log.GetSystemErrors(ctx, since, lines)
	if err != nil {
		return "", err
	}
	return formatJournalEntries(entries), nil
}

func (r *ToolRegistry) logService(ctx context.Context, params map[string]interface{}) (string, error) {
	service := params["service"].(string)
	lines := 0
	if v, ok := params["lines"].(float64); ok {
		lines = int(v)
	}

	entries, err := r.log.GetServiceLogs(ctx, service, lines)
	if err != nil {
		return "", err
	}
	return formatJournalEntries(entries), nil
}

func (r *ToolRegistry) weatherGet(ctx context.Context, params map[string]interface{}) (string, error) {
	location := ""
	if v, ok := params["location"].(string); ok {
		location = v
	}

	weather, err := r.weather.GetWeatherSimple(ctx, location)
	if err != nil {
		return "", err
	}
	return weather, nil
}
