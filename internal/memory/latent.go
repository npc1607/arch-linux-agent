package memory

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// LatentMemory 潜在记忆（向量记忆）
type LatentMemory struct {
	mu         sync.RWMutex
	db         MemoryStore
	embedding  EmbeddingClient
	config     *LatentMemoryConfig
}

// LatentMemoryConfig 配置
type LatentMemoryConfig struct {
	MaxMemories       int           // 最大记忆数
	DefaultImportance float64       // 默认重要性
	DecayRate         float64       // 衰减率
	DecayInterval     time.Duration // 衰减间隔
	EmbeddingModel    string        // Embedding 模型
}

// DefaultLatentMemoryConfig 默认配置
func DefaultLatentMemoryConfig() *LatentMemoryConfig {
	return &LatentMemoryConfig{
		MaxMemories:       10000,
		DefaultImportance: 0.5,
		DecayRate:         0.9,
		DecayInterval:     24 * time.Hour,
		EmbeddingModel:    "text-embedding-3-small",
	}
}

// MemoryStore 记忆存储接口
type MemoryStore interface {
	Store(ctx context.Context, mem *Memory) error
	Retrieve(ctx context.Context, embedding []float32, limit int) ([]*Memory, error)
	GetByID(ctx context.Context, id string) (*Memory, error)
	Update(ctx context.Context, mem *Memory) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filter *MemoryFilter) ([]*Memory, error)
}

// MemoryFilter 记忆过滤器
type MemoryFilter struct {
	Types       []MemoryType
	MinImportance float64
	CreatedAfter  int64
	CreatedBefore int64
	Limit         int
	Offset        int
}

// EmbeddingClient Embedding 客户端接口
type EmbeddingClient interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
	GetDimension() int
}

// NewLatentMemory 创建新的 Latent Memory
func NewLatentMemory(db MemoryStore, embedding EmbeddingClient, config *LatentMemoryConfig) *LatentMemory {
	if config == nil {
		config = DefaultLatentMemoryConfig()
	}
	return &LatentMemory{
		db:        db,
		embedding: embedding,
		config:    config,
	}
}

// FormFromInteraction 从交互中形成记忆
func (lm *LatentMemory) FormFromInteraction(
	ctx context.Context,
	userInput string,
	agentResponse string,
	toolCalls []*ToolCallResult,
) error {

	lm.mu.Lock()
	defer lm.mu.Unlock()

	memories := make([]*Memory, 0)

	// 1. 提取命令执行记忆
	for _, call := range toolCalls {
		if call.Success {
			mem := &Memory{
				Type:    MemoryTypeCommand,
				Content: fmt.Sprintf("Command: %s\nResult: Success", call.Command),
				Metadata: map[string]interface{}{
					"command":   call.Command,
					"exit_code": call.ExitCode,
					"duration":  call.Duration,
					"success":   true,
					"tags":      append([]string{"success"}, call.Tags...),
				},
				CreatedAt:   time.Now().Unix(),
				AccessedAt:  time.Now().Unix(),
				AccessCount: 0,
				Importance:  lm.config.DefaultImportance,
			}
			memories = append(memories, mem)
		} else {
			// 失败的命令，记录为问题记忆
			mem := &Memory{
				Type:    MemoryTypeProblem,
				Content: fmt.Sprintf("Problem: Command '%s' failed\nError: %s\nContext: %s",
					call.Command, call.Error, userInput),
				Metadata: map[string]interface{}{
					"command":  call.Command,
					"exit_code": call.ExitCode,
					"success":  false,
					"tags":     []string{"failed", "troubleshooting"},
				},
				CreatedAt:   time.Now().Unix(),
				AccessedAt:  time.Now().Unix(),
				AccessCount: 0,
				Importance:  lm.config.DefaultImportance + 0.2, // 问题记忆重要性略高
			}
			memories = append(memories, mem)
		}
	}

	// 2. 提取问题解决记忆
	if hasProblemSolving(agentResponse, toolCalls) {
		mem := &Memory{
			Type:    MemoryTypeProblem,
			Content: fmt.Sprintf("Problem: %s\nSolution: %s", userInput, agentResponse),
			Metadata: map[string]interface{}{
				"tags": []string{"troubleshooting", "solved"},
			},
			CreatedAt:   time.Now().Unix(),
			AccessedAt:  time.Now().Unix(),
			AccessCount: 0,
			Importance:  lm.config.DefaultImportance + 0.3,
		}
		memories = append(memories, mem)
	}

	// 3. 生成 Embedding 并存储
	for _, mem := range memories {
		embedding, err := lm.embedding.Embed(ctx, mem.Content)
		if err != nil {
			continue
		}
		mem.Embedding = embedding

		if err := lm.db.Store(ctx, mem); err != nil {
			return fmt.Errorf("failed to store memory: %w", err)
		}
	}

	return nil
}

// hasProblemSolving 判断是否包含问题解决
func hasProblemSolving(response string, toolCalls []*ToolCallResult) bool {
	// 简单判断：如果有工具调用且响应包含解决建议
	if len(toolCalls) > 0 {
		return true
	}
	// TODO: 更复杂的判断逻辑
	return false
}

// Retrieve 检索记忆
func (lm *LatentMemory) Retrieve(
	ctx context.Context,
	query string,
	config *RetrievalConfig,
) ([]*Memory, error) {

	if config == nil {
		config = DefaultRetrievalConfig()
	}

	lm.mu.RLock()
	defer lm.mu.RUnlock()

	switch config.Strategy {
	case RetrieveBySimilarity:
		return lm.retrieveBySimilarity(ctx, query, config.Limit)
	case RetrieveByRecency:
		return lm.retrieveByRecency(ctx, config)
	case RetrieveByImportance:
		return lm.retrieveByImportance(ctx, config)
	case RetrieveHybrid:
		return lm.retrieveHybrid(ctx, query, config)
	default:
		return lm.retrieveHybrid(ctx, query, config)
	}
}

// retrieveBySimilarity 基于相似度检索
func (lm *LatentMemory) retrieveBySimilarity(ctx context.Context, query string, limit int) ([]*Memory, error) {
	embedding, err := lm.embedding.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	memories, err := lm.db.Retrieve(ctx, embedding, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve memories: %w", err)
	}

	// 更新访问统计
	for _, mem := range memories {
		mem.AccessCount++
		mem.AccessedAt = time.Now().Unix()
		lm.db.Update(ctx, mem)
	}

	return memories, nil
}

// retrieveByRecency 基于最近访问检索
func (lm *LatentMemory) retrieveByRecency(ctx context.Context, config *RetrievalConfig) ([]*Memory, error) {
	filter := &MemoryFilter{
		CreatedAfter: time.Now().Unix() - config.TimeRange,
		Limit:        config.Limit,
	}

	return lm.db.List(ctx, filter)
}

// retrieveByImportance 基于重要性检索
func (lm *LatentMemory) retrieveByImportance(ctx context.Context, config *RetrievalConfig) ([]*Memory, error) {
	filter := &MemoryFilter{
		MinImportance: config.MinImportance,
		Limit:         config.Limit,
	}

	return lm.db.List(ctx, filter)
}

// retrieveHybrid 混合检索策略
func (lm *LatentMemory) retrieveHybrid(ctx context.Context, query string, config *RetrievalConfig) ([]*Memory, error) {
	// 1. 获取相似记忆
	similarMemories, err := lm.retrieveBySimilarity(ctx, query, config.Limit*2)
	if err != nil {
		return nil, err
	}

	// 2. 计算混合分数
	for _, mem := range similarMemories {
		recencyScore := calculateRecencyScore(mem.AccessedAt)
		mem.Importance = mem.Importance*0.5 + recencyScore*0.3 + float64(mem.AccessCount)*0.2
	}

	// 3. 按混合分数排序并返回 top N
	return topNMemories(similarMemories, config.Limit), nil
}

// calculateRecencyScore 计算时间衰减分数
func calculateRecencyScore(accessedAt int64) float64 {
	age := time.Now().Unix() - accessedAt
	// 7天内的记忆获得较高分数
	if age < 7*24*3600 {
		return 1.0 - float64(age)/(7*24*3600)
	}
	return 0.1
}

// topNMemories 获取 top N 记忆
func topNMemories(memories []*Memory, n int) []*Memory {
	if len(memories) <= n {
		return memories
	}
	return memories[:n]
}

// InjectToContext 将检索到的记忆注入到 Token Memory
func (lm *LatentMemory) InjectToContext(
	ctx context.Context,
	query string,
	tokenMem *TokenMemory,
	config *RetrievalConfig,
) error {

	if config == nil {
		config = DefaultRetrievalConfig()
	}

	// 检索相关记忆
	memories, err := lm.Retrieve(ctx, query, config)
	if err != nil {
		return err
	}

	if len(memories) == 0 {
		return nil
	}

	// 构建上下文
	contextStr := "[Relevant Memories]\n"
	for _, mem := range memories {
		contextStr += fmt.Sprintf("- [%s] %s\n", mem.Type, mem.Content)
	}

	// 作为系统消息插入到对话历史开头
	tokenMem.Append(Message{
		Role:      "system",
		Content:   contextStr,
		Timestamp: time.Now().Unix(),
	})

	return nil
}

// DecayImportance 重要性衰减
func (lm *LatentMemory) DecayImportance(ctx context.Context) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	filter := &MemoryFilter{
		CreatedAfter: time.Now().Unix() - int64(30*24*3600), // 30天前的记忆
	}

	memories, err := lm.db.List(ctx, filter)
	if err != nil {
		return err
	}

	for _, mem := range memories {
		mem.Importance *= lm.config.DecayRate
		lm.db.Update(ctx, mem)
	}

	return nil
}

// ForgetUnimportant 遗忘不重要的记忆
func (lm *LatentMemory) ForgetUnimportant(ctx context.Context, threshold float64, days int) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	filter := &MemoryFilter{
		MinImportance: threshold,
		CreatedBefore: time.Now().Unix() - int64(days*24*3600),
	}

	memories, err := lm.db.List(ctx, filter)
	if err != nil {
		return err
	}

	for _, mem := range memories {
		lm.db.Delete(ctx, mem.ID)
	}

	return nil
}

// ToolCallResult 工具调用结果
type ToolCallResult struct {
	Command   string
	ExitCode  int
	Duration  int64
	Success   bool
	Output    string
	Error     string
	Tags      []string
}
