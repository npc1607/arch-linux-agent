package memory

import (
	"context"
	"fmt"
	"time"
)

// Manager 记忆管理器，协调 Token Memory 和 Latent Memory
type Manager struct {
	tokenMemory  *TokenMemory
	latentMemory *LatentMemory
	embedding    EmbeddingClient
	config       *ManagerConfig
}

// ManagerConfig 管理器配置
type ManagerConfig struct {
	AutoSummary           bool
	AutoDecay             bool
	DecayInterval         time.Duration
	RetrievalConfig       *RetrievalConfig
	TokenMemoryConfig     *TokenMemoryConfig
	LatentMemoryConfig    *LatentMemoryConfig
}

// DefaultManagerConfig 默认配置
func DefaultManagerConfig() *ManagerConfig {
	return &ManagerConfig{
		AutoSummary:       true,
		AutoDecay:         true,
		DecayInterval:     1 * time.Hour,
		RetrievalConfig:   DefaultRetrievalConfig(),
		TokenMemoryConfig: DefaultTokenMemoryConfig(),
		LatentMemoryConfig: DefaultLatentMemoryConfig(),
	}
}

// NewManager 创建记忆管理器
func NewManager(
	tokenMem *TokenMemory,
	latentMem *LatentMemory,
	embedding EmbeddingClient,
	config *ManagerConfig,
) *Manager {
	if config == nil {
		config = DefaultManagerConfig()
	}
	return &Manager{
		tokenMemory:  tokenMem,
		latentMemory: latentMem,
		embedding:    embedding,
		config:       config,
	}
}

// ProcessInteraction 处理一次交互，形成记忆
func (m *Manager) ProcessInteraction(
	ctx context.Context,
	userInput string,
	agentResponse string,
	toolCalls []*ToolCallResult,
) error {

	// 1. 更新 Token Memory
	m.tokenMemory.Append(Message{
		Role:      "user",
		Content:   userInput,
		Timestamp: time.Now().Unix(),
	})

	m.tokenMemory.Append(Message{
		Role:      "assistant",
		Content:   agentResponse,
		Timestamp: time.Now().Unix(),
	})

	// 更新工作记忆
	if len(toolCalls) > 0 {
		lastCall := toolCalls[len(toolCalls)-1]
		errorMsg := ""
		if !lastCall.Success {
			errorMsg = lastCall.Error
		}
		m.tokenMemory.SetWorkingMemory(lastCall.Command, lastCall.Output, errorMsg)
	}

	// 2. 形成 Latent Memory
	if err := m.latentMemory.FormFromInteraction(ctx, userInput, agentResponse, toolCalls); err != nil {
		return fmt.Errorf("failed to form latent memory: %w", err)
	}

	// 3. 检查是否需要摘要
	if m.config.AutoSummary && m.tokenMemory.NeedsSummary() {
		// TODO: 调用 LLM 进行摘要
		// 这里需要注入 LLM 客户端
	}

	return nil
}

// Query 查询记忆，返回相关上下文
func (m *Manager) Query(ctx context.Context, query string) (string, error) {

	// 1. 从 Latent Memory 检索相关记忆
	if err := m.latentMemory.InjectToContext(
		ctx,
		query,
		m.tokenMemory,
		m.config.RetrievalConfig,
	); err != nil {
		return "", fmt.Errorf("failed to inject latent memory: %w", err)
	}

	// 2. 构建完整上下文
	context := m.tokenMemory.BuildContext()

	return context, nil
}

// GetConversationHistory 获取对话历史
func (m *Manager) GetConversationHistory(n int) []Message {
	return m.tokenMemory.GetRecent(n)
}

// GetWorkingMemory 获取工作记忆
func (m *Manager) GetWorkingMemory() *WorkingMemory {
	return m.tokenMemory.GetWorkingMemory()
}

// SetUserPreference 设置用户偏好
func (m *Manager) SetUserPreference(key, value string) {
	m.tokenMemory.SetUserPreference(key, value)
}

// GetUserPreference 获取用户偏好
func (m *Manager) GetUserPreference(key string) (string, bool) {
	return m.tokenMemory.GetUserPreference(key)
}

// StartMaintenance 启动定期维护任务
func (m *Manager) StartMaintenance(ctx context.Context) {
	ticker := time.NewTicker(m.config.DecayInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.runMaintenance(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// runMaintenance 执行维护任务
func (m *Manager) runMaintenance(ctx context.Context) {
	// 1. Latent Memory 重要性衰减
	if m.config.AutoDecay {
		if err := m.latentMemory.DecayImportance(ctx); err != nil {
			// 记录错误但不中断
			fmt.Printf("Warning: failed to decay importance: %v\n", err)
		}
	}

	// 2. 遗忘不重要的记忆
	if err := m.latentMemory.ForgetUnimportant(ctx, 0.2, 90); err != nil {
		fmt.Printf("Warning: failed to forget unimportant memories: %v\n", err)
	}

	// 3. Token Memory 摘要（如果需要）
	if m.config.AutoSummary && m.tokenMemory.NeedsSummary() {
		// TODO: 实现 LLM 摘要
	}
}

// ClearSession 清空当前会话
func (m *Manager) ClearSession() {
	m.tokenMemory.Clear()
}

// GetStats 获取记忆统计信息
func (m *Manager) GetStats(ctx context.Context) (*MemoryStats, error) {
	stats := &MemoryStats{
		TokenMemoryCount: len(m.tokenMemory.GetAll()),
		TokenCount:       m.tokenMemory.EstimateTokens(),
	}

	// 获取 Latent Memory 统计
	filter := &MemoryFilter{}
	memories, err := m.latentMemory.db.List(ctx, filter)
	if err == nil {
		stats.LatentMemoryCount = len(memories)
	}

	return stats, nil
}

// MemoryStats 记忆统计
type MemoryStats struct {
	TokenMemoryCount  int    `json:"token_memory_count"`
	LatentMemoryCount int    `json:"latent_memory_count"`
	TokenCount        int    `json:"token_count"`
}
