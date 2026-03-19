package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/npc1607/arch-linux-agent/pkg/logger"
	"github.com/sashabaranov/go-openai"
)

// LazyToolConfig 懒加载工具配置
type LazyToolConfig struct {
	Enabled        bool     `json:"enabled"`         // 是否启用懒加载
	TriggerKeywords []string `json:"trigger_keywords"` // 触发关键词
	Description     string   `json:"description"`     // 工具组描述
	Category        string   `json:"category"`        // 分类
}

// ToolLoaderFunc 工具加载函数类型
type ToolLoaderFunc func(ctx context.Context) ([]*Tool, error)

// LazyToolGroup 懒加载工具组
type LazyToolGroup struct {
	Name        string          `json:"name"`
	Config      LazyToolConfig  `json:"config"`
	Loader      ToolLoaderFunc   `json:"-"`
	Loaded      bool            `json:"loaded"`
	Tools       []*Tool         `json:"tools"`
	mu          sync.RWMutex    `json:"-"`
}

// LazyToolRegistry 懒加载工具注册表
type LazyToolRegistry struct {
	registry      *ToolRegistry           // 基础工具注册表
	lazyGroups    map[string]*LazyToolGroup // 懒加载工具组
	config        *LazyLoadConfig
	mu            sync.RWMutex
}

// LazyLoadConfig 懒加载全局配置
type LazyLoadConfig struct {
	Enabled           bool              `json:"enabled"`
	AutoLoad          bool              `json:"auto_load"`           // 是否自动加载匹配的工具
	MaxToolsPerGroup  int               `json:"max_tools_per_group"` // 每组最大工具数
}

// NewLazyToolRegistry 创建懒加载工具注册表
func NewLazyToolRegistry(baseRegistry *ToolRegistry) *LazyToolRegistry {
	return &LazyToolRegistry{
		registry:   baseRegistry,
		lazyGroups: make(map[string]*LazyToolGroup),
		config: &LazyLoadConfig{
			Enabled:          true,
			AutoLoad:         false, // 默认不自动加载，需要用户确认
			MaxToolsPerGroup: 50,
		},
	}
}

// RegisterLazyGroup 注册懒加载工具组
func (r *LazyToolRegistry) RegisterLazyGroup(name string, config LazyToolConfig, loader ToolLoaderFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()

	group := &LazyToolGroup{
		Name:   name,
		Config: config,
		Loader: loader,
		Loaded: false,
		Tools:  make([]*Tool, 0),
	}

	r.lazyGroups[name] = group
	logger.Info("注册懒加载工具组",
		logger.String("group", name),
		logger.String("category", config.Category),
		logger.Strings("keywords", config.TriggerKeywords),
	)
}

// GetToolSummary 获取工具组摘要（不加载工具）
func (r *LazyToolRegistry) GetToolSummary() []openai.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]openai.Tool, 0)

	// 添加已注册的基础工具
	for _, tool := range r.registry.List() {
		tools = append(tools, openai.Tool{
			Type:     "function",
			Function: &tool.Function,
		})
	}

	// 为每个懒加载组添加摘要工具
	for _, group := range r.lazyGroups {
		if !group.Config.Enabled {
			continue
		}

		summaryTool := r.createSummaryTool(group)
		tools = append(tools, openai.Tool{
			Type:     "function",
			Function: summaryTool,
		})
	}

	return tools
}

// createSummaryTool 创建工具组摘要工具
func (r *LazyToolRegistry) createSummaryTool(group *LazyToolGroup) *openai.FunctionDefinition {
	keywordsStr := strings.Join(group.Config.TriggerKeywords, ", ")

	return &openai.FunctionDefinition{
		Name: fmt.Sprintf("load_%s_tools", group.Name),
		Description: fmt.Sprintf(
			"加载 %s 工具组。%s。触发关键词: %s。使用前请确认用户需要使用这些工具。",
			group.Name,
			group.Config.Description,
			keywordsStr,
		),
		Parameters: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{
				"confirm": map[string]interface{}{
					"type":        "boolean",
					"description": "是否确认加载此工具组",
				},
			},
			"required": []string{"confirm"},
		},
	}
}

// LoadGroup 加载指定的工具组
func (r *LazyToolRegistry) LoadGroup(ctx context.Context, groupName string, autoConfirm bool) ([]*Tool, error) {
	r.mu.Lock()
	group, exists := r.lazyGroups[groupName]
	r.mu.Unlock()

	if !exists {
		return nil, fmt.Errorf("工具组不存在: %s", groupName)
	}

	group.mu.Lock()
	defer group.mu.Unlock()

	// 如果已经加载，直接返回
	if group.Loaded {
		logger.Info("工具组已加载", logger.String("group", groupName))
		return group.Tools, nil
	}

	// 检查是否需要用户确认
	if !autoConfirm && !r.config.AutoLoad {
		logger.Info("等待用户确认加载工具组", logger.String("group", groupName))
		return nil, &ToolLoadPendingError{
			GroupName: groupName,
			Message:   fmt.Sprintf("需要加载 %s 工具组来执行此操作", group.Config.Description),
		}
	}

	logger.Info("开始加载工具组", logger.String("group", groupName))

	// 调用加载函数
	tools, err := group.Loader(ctx)
	if err != nil {
		return nil, fmt.Errorf("加载工具组失败: %w", err)
	}

	// 限制工具数量
	if r.config.MaxToolsPerGroup > 0 && len(tools) > r.config.MaxToolsPerGroup {
		tools = tools[:r.config.MaxToolsPerGroup]
		logger.Warn("工具数量超过限制，已截断",
			logger.Int("max", r.config.MaxToolsPerGroup),
			logger.Int("original", len(tools)),
		)
	}

	// 注册工具到基础注册表
	for _, tool := range tools {
		r.registry.Register(tool)
	}

	group.Tools = tools
	group.Loaded = true

	logger.Info("工具组加载完成",
		logger.String("group", groupName),
		logger.Int("count", len(tools)),
	)

	return tools, nil
}

// CheckAndLoad 检查消息并自动加载相关工具
func (r *LazyToolRegistry) CheckAndLoad(ctx context.Context, message string) (loadedGroups []string, err error) {
	// 先用读锁检查哪些工具组需要加载
	r.mu.RLock()
	groupsToLoad := make([]string, 0)
	messageLower := strings.ToLower(message)

	for _, group := range r.lazyGroups {
		if !group.Config.Enabled || group.Loaded {
			continue
		}

		// 检查是否匹配触发关键词
		for _, keyword := range group.Config.TriggerKeywords {
			if strings.Contains(messageLower, strings.ToLower(keyword)) {
				groupsToLoad = append(groupsToLoad, group.Name)
				break
			}
		}
	}
	r.mu.RUnlock()

	// 释放读锁后，再加载工具组（避免死锁）
	loadedGroups = make([]string, 0)
	for _, groupName := range groupsToLoad {
		_, err := r.LoadGroup(ctx, groupName, r.config.AutoLoad)
		if err != nil {
			if IsToolLoadPending(err) {
				// 返回待加载状态，让上层处理
				return loadedGroups, err
			}
			return loadedGroups, err
		}
		loadedGroups = append(loadedGroups, groupName)
	}

	return loadedGroups, nil
}

// GetLoadedTools 获取所有已加载的工具
func (r *LazyToolRegistry) GetLoadedTools() []openai.Tool {
	// 获取基础工具
	tools := r.registry.GetOpenAITools()

	return tools
}

// GetGroupStatus 获取工具组状态
func (r *LazyToolRegistry) GetGroupStatus() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	status := make(map[string]interface{})
	for name, group := range r.lazyGroups {
		status[name] = map[string]interface{}{
			"loaded":       group.Loaded,
			"description":  group.Config.Description,
			"category":     group.Config.Category,
			"tool_count":   len(group.Tools),
			"keywords":     group.Config.TriggerKeywords,
		}
	}

	return status
}

// IsGroupLoaded 检查工具组是否已加载
func (r *LazyToolRegistry) IsGroupLoaded(groupName string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	group, exists := r.lazyGroups[groupName]
	if !exists {
		return false
	}

	group.mu.RLock()
	defer group.mu.RUnlock()

	return group.Loaded
}

// ToolLoadPendingError 工具加载待处理错误
type ToolLoadPendingError struct {
	GroupName string
	Message   string
}

func (e *ToolLoadPendingError) Error() string {
	return e.Message
}

// IsToolLoadPending 检查是否是工具加载待处理错误
func IsToolLoadPending(err error) bool {
	_, ok := err.(*ToolLoadPendingError)
	return ok
}

// GetGroupNameFromError 从错误中获取工具组名
func GetGroupNameFromError(err error) (string, bool) {
	if loadErr, ok := err.(*ToolLoadPendingError); ok {
		return loadErr.GroupName, true
	}
	return "", false
}
