package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
)

// InMemoryStore 内存存储实现（用于开发和测试）
type InMemoryStore struct {
	mu       sync.RWMutex
	memories map[string]*Memory
}

// NewInMemoryStore 创建内存存储
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		memories: make(map[string]*Memory),
	}
}

// Store 存储记忆
func (s *InMemoryStore) Store(ctx context.Context, mem *Memory) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if mem.ID == "" {
		mem.ID = uuid.New().String()
	}

	s.memories[mem.ID] = mem
	return nil
}

// Retrieve 检索记忆（基于向量相似度）
func (s *InMemoryStore) Retrieve(ctx context.Context, embedding []float32, limit int) ([]*Memory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.memories) == 0 {
		return []*Memory{}, nil
	}

	// 计算相似度并排序
	type ScoredMemory struct {
		Memory *Memory
		Score  float64
	}

	scored := make([]ScoredMemory, 0, len(s.memories))
	for _, mem := range s.memories {
		if len(mem.Embedding) == 0 {
			continue
		}
		score := cosineSimilarity(embedding, mem.Embedding)
		scored = append(scored, ScoredMemory{
			Memory: mem,
			Score:  score,
		})
	}

	// 按相似度排序
	for i := 0; i < len(scored); i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].Score > scored[i].Score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	// 返回 top N
	result := make([]*Memory, 0, limit)
	for i := 0; i < len(scored) && i < limit; i++ {
		result = append(result, scored[i].Memory)
	}

	return result, nil
}

// GetByID 根据 ID 获取记忆
func (s *InMemoryStore) GetByID(ctx context.Context, id string) (*Memory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	mem, ok := s.memories[id]
	if !ok {
		return nil, fmt.Errorf("memory not found: %s", id)
	}
	return mem, nil
}

// Update 更新记忆
func (s *InMemoryStore) Update(ctx context.Context, mem *Memory) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.memories[mem.ID]; !ok {
		return fmt.Errorf("memory not found: %s", mem.ID)
	}

	s.memories[mem.ID] = mem
	return nil
}

// Delete 删除记忆
func (s *InMemoryStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.memories, id)
	return nil
}

// List 列出记忆
func (s *InMemoryStore) List(ctx context.Context, filter *MemoryFilter) ([]*Memory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Memory, 0)

	for _, mem := range s.memories {
		// 应用过滤器
		if filter != nil {
			// 类型过滤
			if len(filter.Types) > 0 {
				typeMatch := false
				for _, t := range filter.Types {
					if mem.Type == t {
						typeMatch = true
						break
					}
				}
				if !typeMatch {
					continue
				}
			}

			// 重要性过滤
			if filter.MinImportance > 0 && mem.Importance < filter.MinImportance {
				continue
			}

			// 时间过滤
			if filter.CreatedAfter > 0 && mem.CreatedAt < filter.CreatedAfter {
				continue
			}
			if filter.CreatedBefore > 0 && mem.CreatedAt > filter.CreatedBefore {
				continue
			}
		}

		result = append(result, mem)
	}

	// 应用 limit 和 offset
	if filter != nil {
		if filter.Offset > 0 && filter.Offset < len(result) {
			result = result[filter.Offset:]
		}
		if filter.Limit > 0 && filter.Limit < len(result) {
			result = result[:filter.Limit]
		}
	}

	return result, nil
}

// cosineSimilarity 计算余弦相似度
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct float64
	var normA float64
	var normB float64

	for i := 0; i < len(a); i++ {
		dotProduct += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (sqrt(normA) * sqrt(normB))
}

// sqrt 简单的平方根实现
func sqrt(x float64) float64 {
	// 牛顿迭代法
	z := 1.0
	for i := 0; i < 10; i++ {
		z -= (z*z - x) / (2 * z)
	}
	return z
}

// Count 返回记忆总数
func (s *InMemoryStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.memories)
}

// Clear 清空所有记忆
func (s *InMemoryStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.memories = make(map[string]*Memory)
}
