package memory

// MemoryType 定义记忆类型
type MemoryType string

const (
	MemoryTypeSystem     MemoryType = "system"     // 系统知识
	MemoryTypeCommand    MemoryType = "command"    // 命令执行记录
	MemoryTypeProblem    MemoryType = "problem"    // 问题解决
	MemoryTypePreference MemoryType = "preference" // 用户偏好
	MemoryTypePattern    MemoryType = "pattern"    // 操作模式
)

// Message 对话消息
type Message struct {
	Role      string                 `json:"role"`
	Content   string                 `json:"content"`
	Timestamp int64                  `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Memory 记忆结构
type Memory struct {
	ID          string                 `json:"id"`
	Type        MemoryType             `json:"type"`
	Content     string                 `json:"content"`
	Embedding   []float32              `json:"embedding,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   int64                  `json:"created_at"`
	AccessedAt  int64                  `json:"accessed_at"`
	AccessCount int                    `json:"access_count"`
	Importance  float64                `json:"importance"`
}

// Metadata 命令元数据
type CommandMetadata struct {
	Command   string        `json:"command"`
	ExitCode  int           `json:"exit_code"`
	Duration  int64         `json:"duration"` // 毫秒
	Success   bool          `json:"success"`
	Output    string        `json:"output,omitempty"`
	Error     string        `json:"error,omitempty"`
	Tags      []string      `json:"tags,omitempty"`
	Context   map[string]string `json:"context,omitempty"`
}

// RetrievalStrategy 检索策略
type RetrievalStrategy int

const (
	RetrieveBySimilarity  RetrievalStrategy = iota // 语义相似度
	RetrieveByRecency                               // 时间最近
	RetrieveByImportance                            // 重要性
	RetrieveHybrid                                 // 混合策略
)

// RetrievalConfig 检索配置
type RetrievalConfig struct {
	Strategy      RetrievalStrategy
	Limit         int
	MinImportance float64
	TimeRange     int64 // 时间范围（秒）
	Types         []MemoryType
}

// DefaultRetrievalConfig 默认检索配置
func DefaultRetrievalConfig() *RetrievalConfig {
	return &RetrievalConfig{
		Strategy:      RetrieveHybrid,
		Limit:         5,
		MinImportance: 0.3,
		TimeRange:     7 * 24 * 3600, // 7天
	}
}
