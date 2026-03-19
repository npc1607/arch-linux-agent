package memory

import (
	"fmt"
	"sync"
	"time"
)

// TokenMemory 令牌级记忆（上下文窗口记忆）
type TokenMemory struct {
	mu                sync.RWMutex
	ConversationHistory []Message
	SessionContext    *SessionContext
	WorkingMemory     *WorkingMemory
	config            *TokenMemoryConfig
}

// TokenMemoryConfig 配置
type TokenMemoryConfig struct {
	MaxHistory        int     // 最大保留消息数
	MaxTokens         int     // 最大 token 数
	SummaryThreshold  int     // 触发摘要的消息数
	KeepRecent        int     // 摘要后保留的消息数
}

// DefaultTokenMemoryConfig 默认配置
func DefaultTokenMemoryConfig() *TokenMemoryConfig {
	return &TokenMemoryConfig{
		MaxHistory:       50,
		MaxTokens:        8000,
		SummaryThreshold: 30,
		KeepRecent:       10,
	}
}

// SessionContext 会话上下文
type SessionContext struct {
	CurrentTask      string                 `json:"current_task"`
	PendingActions   []string               `json:"pending_actions"`
	SystemState      map[string]interface{} `json:"system_state"`
	UserPreferences  map[string]string      `json:"user_preferences"`
	StartTime        int64                  `json:"start_time"`
}

// WorkingMemory 工作记忆（临时存储）
type WorkingMemory struct {
	LastCommand   string   `json:"last_command"`
	LastOutput    string   `json:"last_output"`
	LastError     string   `json:"last_error"`
	ErrorHistory  []string `json:"error_history"`
	UpdatedTime   int64    `json:"updated_time"`
}

// NewTokenMemory 创建新的 Token Memory
func NewTokenMemory(config *TokenMemoryConfig) *TokenMemory {
	if config == nil {
		config = DefaultTokenMemoryConfig()
	}
	return &TokenMemory{
		ConversationHistory: make([]Message, 0),
		SessionContext: &SessionContext{
			SystemState:     make(map[string]interface{}),
			UserPreferences: make(map[string]string),
			StartTime:       time.Now().Unix(),
		},
		WorkingMemory: &WorkingMemory{
			ErrorHistory: make([]string, 0),
		},
		config: config,
	}
}

// Append 添加消息到历史
func (tm *TokenMemory) Append(msg Message) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	msg.Timestamp = time.Now().Unix()
	tm.ConversationHistory = append(tm.ConversationHistory, msg)

	// 检查是否需要摘要
	if len(tm.ConversationHistory) > tm.config.MaxHistory {
		tm.trimHistory()
	}
}

// GetRecent 获取最近 N 条消息
func (tm *TokenMemory) GetRecent(n int) []Message {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if n <= 0 || n > len(tm.ConversationHistory) {
		n = len(tm.ConversationHistory)
	}
	return tm.ConversationHistory[len(tm.ConversationHistory)-n:]
}

// GetAll 获取所有消息
func (tm *TokenMemory) GetAll() []Message {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	result := make([]Message, len(tm.ConversationHistory))
	copy(result, tm.ConversationHistory)
	return result
}

// Clear 清空历史
func (tm *TokenMemory) Clear() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.ConversationHistory = make([]Message, 0)
}

// trimHistory 裁剪历史（当超过最大长度时）
func (tm *TokenMemory) trimHistory() {
	// 保留最近的消息
	keep := tm.config.KeepRecent
	if keep > len(tm.ConversationHistory) {
		keep = len(tm.ConversationHistory)
	}
	tm.ConversationHistory = tm.ConversationHistory[len(tm.ConversationHistory)-keep:]
}

// NeedsSummary 检查是否需要摘要
func (tm *TokenMemory) NeedsSummary() bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return len(tm.ConversationHistory) >= tm.config.SummaryThreshold
}

// SetWorkingMemory 设置工作记忆
func (tm *TokenMemory) SetWorkingMemory(command, output, errorMsg string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.WorkingMemory.LastCommand = command
	tm.WorkingMemory.LastOutput = output
	tm.WorkingMemory.LastError = errorMsg
	tm.WorkingMemory.UpdatedTime = time.Now().Unix()

	if errorMsg != "" {
		tm.WorkingMemory.ErrorHistory = append(tm.WorkingMemory.ErrorHistory, errorMsg)
		// 只保留最近 10 个错误
		if len(tm.WorkingMemory.ErrorHistory) > 10 {
			tm.WorkingMemory.ErrorHistory = tm.WorkingMemory.ErrorHistory[len(tm.WorkingMemory.ErrorHistory)-10:]
		}
	}
}

// GetWorkingMemory 获取工作记忆
func (tm *TokenMemory) GetWorkingMemory() *WorkingMemory {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	// 返回副本
	wm := *tm.WorkingMemory
	errors := make([]string, len(wm.ErrorHistory))
	copy(errors, wm.ErrorHistory)
	wm.ErrorHistory = errors
	return &wm
}

// UpdateSessionContext 更新会话上下文
func (tm *TokenMemory) UpdateSessionContext(task string, actions []string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.SessionContext.CurrentTask = task
	tm.SessionContext.PendingActions = actions
}

// SetUserPreference 设置用户偏好
func (tm *TokenMemory) SetUserPreference(key, value string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.SessionContext.UserPreferences[key] = value
}

// GetUserPreference 获取用户偏好
func (tm *TokenMemory) GetUserPreference(key string) (string, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	val, ok := tm.SessionContext.UserPreferences[key]
	return val, ok
}

// BuildContext 构建完整的上下文字符串（用于 LLM）
func (tm *TokenMemory) BuildContext() string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	context := ""

	// 添加工作记忆
	if tm.WorkingMemory.LastCommand != "" {
		context += "[Recent Activity]\n"
		context += fmt.Sprintf("Last Command: %s\n", tm.WorkingMemory.LastCommand)
		if tm.WorkingMemory.LastError != "" {
			context += fmt.Sprintf("Last Error: %s\n", tm.WorkingMemory.LastError)
		}
		context += "\n"
	}

	// 添加会话上下文
	if tm.SessionContext.CurrentTask != "" {
		context += "[Current Task]\n"
		context += tm.SessionContext.CurrentTask + "\n\n"
	}

	// 添加对话历史
	context += "[Conversation History]\n"
	for _, msg := range tm.ConversationHistory {
		context += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
	}

	return context
}

// EstimateTokens 估算当前 token 数（粗略估算：中文 1.5 token/字符，英文 4 token/单词）
func (tm *TokenMemory) EstimateTokens() int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	totalChars := 0
	for _, msg := range tm.ConversationHistory {
		totalChars += len(msg.Content)
	}
	// 粗略估算：平均 2 字符/token
	return totalChars / 2
}
