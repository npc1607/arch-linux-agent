package memory

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

// OpenAIEmbeddingClient OpenAI Embedding 客户端
type OpenAIEmbeddingClient struct {
	client *openai.Client
	model  string
	dim    int
}

// NewOpenAIEmbeddingClient 创建 OpenAI Embedding 客户端
func NewOpenAIEmbeddingClient(apiKey, model string) (*OpenAIEmbeddingClient, error) {
	if model == "" {
		model = openai.Embedding3Small
	}

	client := openai.NewClient(apiKey)

	dim := 1536 // text-embedding-3-small 默认维度
	if model == openai.Embedding3Large {
		dim = 3072
	} else if model == openai.EmbeddingAda002 {
		dim = 1536
	}

	return &OpenAIEmbeddingClient{
		client: client,
		model:  model,
		dim:    dim,
	}, nil
}

// Embed 生成单个文本的 Embedding
func (c *OpenAIEmbeddingClient) Embed(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("empty text")
	}

	resp, err := c.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: []string{text},
		Model: c.model,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}

	return resp.Data[0].Embedding, nil
}

// EmbedBatch 批量生成 Embedding
func (c *OpenAIEmbeddingClient) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	// OpenAI 支持批量处理
	resp, err := c.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: texts,
		Model: c.model,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create embeddings: %w", err)
	}

	embeddings := make([][]float32, len(resp.Data))
	for i, data := range resp.Data {
		embeddings[i] = data.Embedding
	}

	return embeddings, nil
}

// GetDimension 获取 Embedding 维度
func (c *OpenAIEmbeddingClient) GetDimension() int {
	return c.dim
}

// MockEmbeddingClient 模拟 Embedding 客户端（用于测试）
type MockEmbeddingClient struct {
	dimension int
}

// NewMockEmbeddingClient 创建模拟 Embedding 客户端
func NewMockEmbeddingClient(dimension int) *MockEmbeddingClient {
	return &MockEmbeddingClient{dimension: dimension}
}

// Embed 生成模拟 Embedding
func (c *MockEmbeddingClient) Embed(ctx context.Context, text string) ([]float32, error) {
	// 基于文本生成确定性的伪随机向量
	embedding := make([]float32, c.dimension)
	hash := simpleHash(text)
	for i := 0; i < c.dimension; i++ {
		embedding[i] = float32((hash*uint32(i))%1000) / 1000.0
	}
	return embedding, nil
}

// EmbedBatch 批量生成模拟 Embedding
func (c *MockEmbeddingClient) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		embed, _ := c.Embed(ctx, text)
		embeddings[i] = embed
	}
	return embeddings, nil
}

// GetDimension 获取维度
func (c *MockEmbeddingClient) GetDimension() int {
	return c.dimension
}

// simpleHash 简单的哈希函数
func simpleHash(s string) uint32 {
	hash := uint32(2166136261)
	for i := 0; i < len(s); i++ {
		hash ^= uint32(s[i])
		hash *= 16777619
	}
	return hash
}
