package llm

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/npc1607/arch-linux-agent/internal/config"
	"github.com/npc1607/arch-linux-agent/pkg/logger"
	"github.com/sashabaranov/go-openai"
)

// Client LLM 客户端
type Client struct {
	client    *openai.Client
	config    *config.LLMConfig
	model     string
	baseURL   string
}

// Message 消息
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Function 函数定义
type Function struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ToolCall 工具调用
type ToolCall struct {
	ID       string
	Name     string
	Arguments string
}

// Response 响应
type Response struct {
	Content   string
	ToolCalls []ToolCall
	Usage     openai.Usage
}

// NewClient 创建 LLM 客户端
func NewClient(cfg *config.LLMConfig) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("配置为空")
	}

	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API Key 未设置")
	}

	clientConfig := openai.DefaultConfig(cfg.APIKey)

	// 设置自定义 Base URL
	if cfg.BaseURL != "" {
		clientConfig.BaseURL = cfg.BaseURL
	}

	client := openai.NewClientWithConfig(clientConfig)

	return &Client{
		client:  client,
		config:  cfg,
		model:   cfg.Model,
		baseURL: cfg.BaseURL,
	}, nil
}

// Chat 对话（非流式）
func (c *Client) Chat(ctx context.Context, messages []Message, functions []Function) (*Response, error) {
	logger.Debug("LLM 请求",
		logger.String("model", c.model),
		logger.Int("messages", len(messages)),
	)

	// 转换消息格式
	chatMessages := c.convertMessages(messages)

	// 创建请求
	request := openai.ChatCompletionRequest{
		Model:       c.model,
		Messages:    chatMessages,
		MaxTokens:   c.config.MaxTokens,
		Temperature: float32(c.config.Temperature),
	}

	// 添加函数
	if len(functions) > 0 {
		request.Tools = c.convertFunctions(functions)
		request.ToolChoice = "auto"
	}

	// 发送请求
	resp, err := c.client.CreateChatCompletion(ctx, request)
	if err != nil {
		logger.Error("LLM 请求失败", logger.Err(err))
		return nil, fmt.Errorf("LLM 请求失败: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("未收到响应")
	}

	choice := resp.Choices[0]

	// 解析响应
	response := &Response{
		Content: choice.Message.Content,
		Usage:   resp.Usage,
	}

	// 解析工具调用
	if len(choice.Message.ToolCalls) > 0 {
		response.ToolCalls = make([]ToolCall, len(choice.Message.ToolCalls))
		for i, tc := range choice.Message.ToolCalls {
			response.ToolCalls[i] = ToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			}
		}
	}

	logger.Debug("LLM 响应",
		logger.String("content", response.Content),
		logger.Int("tool_calls", len(response.ToolCalls)),
		logger.Any("usage", resp.Usage),
	)

	return response, nil
}

// ChatStream 对话（流式）
func (c *Client) ChatStream(ctx context.Context, messages []Message, functions []Function, callback func(content string)) error {
	logger.Debug("LLM 流式请求",
		logger.String("model", c.model),
		logger.Int("messages", len(messages)),
	)

	// 转换消息格式
	chatMessages := c.convertMessages(messages)

	// 创建请求
	request := openai.ChatCompletionRequest{
		Model:       c.model,
		Messages:    chatMessages,
		MaxTokens:   c.config.MaxTokens,
		Temperature: float32(c.config.Temperature),
		Stream:      true,
	}

	// 添加函数
	if len(functions) > 0 {
		request.Tools = c.convertFunctions(functions)
		request.ToolChoice = "auto"
	}

	// 创建流式请求
	stream, err := c.client.CreateChatCompletionStream(ctx, request)
	if err != nil {
		logger.Error("LLM 流式请求失败", logger.Err(err))
		return fmt.Errorf("LLM 流式请求失败: %w", err)
	}
	defer stream.Close()

	var fullContent strings.Builder

	for {
		response, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			logger.Error("LLM 流式响应读取失败", logger.Err(err))
			return fmt.Errorf("读取流式响应失败: %w", err)
		}

		if len(response.Choices) == 0 {
			continue
		}

		delta := response.Choices[0].Delta.Content
		if delta != "" {
			fullContent.WriteString(delta)
			if callback != nil {
				callback(delta)
			}
		}
	}

	logger.Debug("LLM 流式响应完成",
		logger.Int("content_length", fullContent.Len()),
	)

	return nil
}

// GetModel 获取当前模型
func (c *Client) GetModel() string {
	return c.model
}

// SetModel 设置模型
func (c *Client) SetModel(model string) {
	c.model = model
}

// convertMessages 转换消息格式
func (c *Client) convertMessages(messages []Message) []openai.ChatCompletionMessage {
	result := make([]openai.ChatCompletionMessage, len(messages))
	for i, msg := range messages {
		result[i] = openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	return result
}

// convertFunctions 转换函数格式
func (c *Client) convertFunctions(functions []Function) []openai.Tool {
	tools := make([]openai.Tool, len(functions))
	for i, fn := range functions {
		tools[i] = openai.Tool{
			Type: "function",
			Function: &openai.FunctionDefinition{
				Name:        fn.Name,
				Description: fn.Description,
				Parameters:  fn.Parameters,
			},
		}
	}
	return tools
}

// EmbeddingClient Embedding 客户端
type EmbeddingClient struct {
	client *openai.Client
	model  string
	dim    int
}

// NewEmbeddingClient 创建 Embedding 客户端
func NewEmbeddingClient(apiKey string) (*EmbeddingClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API Key 未设置")
	}

	client := openai.NewClient(apiKey)

	return &EmbeddingClient{
		client: client,
		model:  string(openai.SmallEmbedding3),
		dim:    1536,
	}, nil
}

// Embed 生成 Embedding
func (c *EmbeddingClient) Embed(ctx context.Context, text string) ([]float32, error) {
	resp, err := c.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: []string{text},
		Model: openai.EmbeddingModel(c.model),
	})
	if err != nil {
		return nil, fmt.Errorf("创建 Embedding 失败: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("未收到 Embedding 响应")
	}

	return resp.Data[0].Embedding, nil
}

// EmbedBatch 批量生成 Embedding
func (c *EmbeddingClient) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	resp, err := c.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: texts,
		Model: openai.EmbeddingModel(c.model),
	})
	if err != nil {
		return nil, fmt.Errorf("批量创建 Embedding 失败: %w", err)
	}

	embeddings := make([][]float32, len(resp.Data))
	for i, data := range resp.Data {
		embeddings[i] = data.Embedding
	}

	return embeddings, nil
}

// GetDimension 获取 Embedding 维度
func (c *EmbeddingClient) GetDimension() int {
	return c.dim
}
