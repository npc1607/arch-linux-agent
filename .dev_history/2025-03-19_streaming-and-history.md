# 流式输出和对话历史功能实现

**完成时间**: 2025-03-19 18:30
**任务类型**: 功能增强
**状态**: ✅ 已完成

---

## 📋 任务概述

实现两个核心用户体验功能：
1. **流式输出** - 实时显示 AI 响应，逐 token 输出
2. **对话历史管理** - 查看、管理对话历史记录

---

## 🏗️ 实现详情

### 1. 流式输出实现

#### 1.1 LLM Client 层 (`internal/llm/client.go`)

已存在的 `ChatStream()` 方法：
```go
func (c *Client) ChatStream(
    ctx context.Context,
    messages []Message,
    functions []Function,
    callback func(content string),
) error
```

**特性**：
- 使用 OpenAI 流式 API
- 逐 token 回调
- 自动错误处理
- EOF 检测

#### 1.2 Agent 层 (`internal/agent/agent.go`)

**新增方法**：

```go
// ProcessStream 处理用户输入（流式输出）
func (a *Agent) ProcessStream(
    ctx context.Context,
    userInput string,
    streamCallback func(string),
) error {
    // 1. 添加用户消息到历史
    // 2. 构建消息列表和工具列表
    // 3. 调用 LLM 流式接口
    // 4. 实时回调每个 token
    // 5. 保存完整响应到历史
}
```

**实现细节**：
- 使用 `strings.Builder` 高效拼接完整响应
- 实时调用 streamCallback 输出 token
- 完整响应保存到 messages 历史

#### 1.3 ChatSession 层 (`internal/chat/session.go`)

**新增方法**：

```go
// ProcessStream 处理用户输入（流式输出）
func (s *ChatSession) ProcessStream(
    ctx context.Context,
    userInput string,
) error {
    fmt.Print("🤖 Agent> ")

    fullResponse := strings.Builder{}
    err := s.agent.ProcessStream(ctx, userInput, func(chunk string) {
        fmt.Print(chunk)           // 实时输出
        fullResponse.WriteString(chunk)
    })

    // 确保以换行结束
    if fullResponse.Len() > 0 {
        lastChar := fullResponse.String()[len(fullResponse.String())-1]
        if lastChar != '\n' {
            fmt.Println()
        }
    }

    return err
}
```

**用户体验优化**：
- 先输出 "🤖 Agent> " 提示符
- 实时打印每个 token
- 自动处理换行

#### 1.4 CLI 层 (`cmd/chat.go`)

**运行时模式选择**：

```go
// Process user input
var processErr error
if stream {
    // 使用流式输出
    processErr = session.ProcessStream(ctx, input)
} else {
    // 使用普通输出
    processErr = session.Process(ctx, input)
}
```

**新增 stream 命令**：
```go
// Handle stream toggle command
if input == "stream" {
    stream = !stream
    mode := "禁用"
    if stream {
        mode = "启用"
    }
    fmt.Fprintf(os.Stderr, "流式输出已%s\n", mode)
    continue
}
```

---

### 2. 对话历史实现

#### 2.1 Agent 层 (`internal/agent/agent.go`)

已存在的方法：
```go
// GetHistory 获取对话历史
func (a *Agent) GetHistory() []llm.Message {
    return a.messages
}

// ClearHistory 清空对话历史
func (a *Agent) ClearHistory() {
    a.messages = make([]llm.Message, 0)
}
```

#### 2.2 ChatSession 层 (`internal/chat/session.go`)

**新增方法**：

```go
// GetHistory 获取对话历史
func (s *ChatSession) GetHistory() []llm.Message {
    history := s.agent.GetHistory()
    if history == nil {
        return []llm.Message{}
    }
    return history
}

// GetHistoryFormatted 获取格式化的对话历史（用于显示）
func (s *ChatSession) GetHistoryFormatted() []string {
    history := s.agent.GetHistory()
    result := make([]string, len(history))

    for i, msg := range history {
        role := msg.Role
        roleDisplay := role
        switch role {
        case "user":
            roleDisplay = "👤 You"
        case "assistant":
            roleDisplay = "🤖 Agent"
        case "tool":
            roleDisplay = "🔧 Tool"
        }
        result[i] = fmt.Sprintf("%s> %s", roleDisplay, msg.Content)
    }

    return result
}
```

**特点**：
- `GetHistory()` 返回原始消息数组
- `GetHistoryFormatted()` 返回格式化字符串数组
- 使用表情符号标识角色类型

#### 2.3 CLI 层 (`cmd/chat.go`)

**新增 history 命令**：
```go
// Handle history command
if input == "history" || input == "hist" {
    printHistory(session)
    continue
}
```

**显示函数**：
```go
func printHistory(session *chat.ChatSession) {
    history := session.GetHistoryFormatted()
    if len(history) == 0 {
        fmt.Fprintln(os.Stderr, "暂无对话历史")
        return
    }

    fmt.Fprintln(os.Stderr, "\n=== 对话历史 ===")
    for i, msg := range history {
        fmt.Fprintf(os.Stderr, "%d. %s\n", i+1, msg)
    }
    fmt.Fprintln(os.Stderr, "================\n")
}
```

**已存在的 clear 命令**：
```go
// Handle clear command
if input == "clear" || input == "cls" {
    session.ClearHistory()
    fmt.Fprintln(os.Stderr, "对话历史已清空")
    continue
}
```

---

### 3. Agent 双模式支持

#### 3.1 模式切换机制

```go
type Agent struct {
    // ... 其他字段
    usePlanner bool  // 是否使用规划模式
}
```

```go
func (a *Agent) Process(...) (string, error) {
    if a.usePlanner {
        return a.processWithPlanner(ctx, userInput, streamCallback)
    }
    return a.processWithFunctionCalling(ctx, userInput, streamCallback)
}
```

#### 3.2 Function Calling 模式

```go
func (a *Agent) processWithFunctionCalling(...) (string, error) {
    // 1. 构建消息和工具列表
    // 2. 调用 LLM（带工具定义）
    // 3. 如果有工具调用，执行工具
    // 4. 让 LLM 总结工具结果
}
```

#### 3.3 规划模式 (Planner Mode)

```go
func (a *Agent) processWithPlanner(...) (string, error) {
    // 1. 任务规划
    plan, err := a.planner.Plan(ctx, userInput)

    // 2. 执行计划步骤
    for _, step := range plan.Steps {
        result, err := a.planner.ExecuteStep(ctx, step)
        // ...
    }

    // 3. 构建响应
    response := a.buildResponseFromPlan(...)
}
```

---

## 📊 代码统计

| 文件 | 行数 | 功能 |
|------|------|------|
| `internal/llm/client.go` | 203 | LLM Client（已存在） |
| `internal/agent/agent.go` | 416 | Agent 核心逻辑 |
| `internal/chat/session.go` | 177 | Chat Session（重构） |
| `cmd/chat.go` | 280+ | CLI 对话命令 |
| **总计** | **1076+** | - |

---

## 🎨 设计亮点

### 1. 流式输出的分层设计

```
┌─────────────────────────────────────────────────┐
│ CLI Layer (chat.go)                             │
│ - 根据 stream 标志选择模式                       │
│ - 提供 stream 命令切换模式                       │
└────────────────┬────────────────────────────────┘
                 │
┌────────────────▼────────────────────────────────┐
│ Session Layer (session.go)                      │
│ - ProcessStream() 实时输出 token                │
│ - 自动处理换行和格式                            │
└────────────────┬────────────────────────────────┘
                 │
┌────────────────▼────────────────────────────────┐
│ Agent Layer (agent.go)                          │
│ - ProcessStream() 调用 LLM                      │
│ - 管理对话历史                                  │
└────────────────┬────────────────────────────────┘
                 │
┌────────────────▼────────────────────────────────┐
│ LLM Layer (client.go)                           │
│ - ChatStream() API 调用                         │
│ - Token 级回调                                  │
└─────────────────────────────────────────────────┘
```

### 2. 对话历史的两种格式

**原始格式**（用于程序访问）：
```go
[]llm.Message{
    {Role: "user", Content: "检查系统"},
    {Role: "assistant", Content: "系统正常..."},
}
```

**格式化格式**（用于显示）：
```go
[]string{
    "👤 You> 检查系统",
    "🤖 Agent> 系统正常...",
}
```

### 3. 运行时模式切换

用户可以在对话过程中动态切换流式输出：
```
🤖 You> stream
流式输出已启用

🤖 You> 介绍自己
🤖 Agent> 我是...（逐字输出）

🤖 You> stream
流式输出已禁用

🤖 You> 介绍自己
🤖 Agent> 我是...（等待完整响应后一次性输出）
```

### 4. 性能优化

- **strings.Builder**: 高效拼接字符串
- **实时输出**: 减少等待时间，提升体验
- **历史管理**: 使用切片存储，访问 O(1)

---

## 🚀 使用示例

### 流式输出

```bash
# 启用流式输出（默认）
./arch-agent chat

🤖 You> 检查系统状态
🤖 Agent> 正在检查系统...

=== CPU ===
CPU 使用率: 12.5%
...
```

```bash
# 禁用流式输出
./arch-agent chat --no-stream

🤖 You> 检查系统状态
（等待...）
🤖 Agent> 正在检查系统...

=== CPU ===
CPU 使用率: 12.5%
...
```

### 对话历史

```bash
🤖 You> 检查内存
🤖 Agent> 总内存: 16.0G...

🤖 You> 检查 CPU
🤖 Agent> CPU 使用率: 12.5%...

🤖 You> history

=== 对话历史 ===
1. 👤 You> 检查内存
2. 🤖 Agent> 总内存: 16.0G...
3. 👤 You> 检查 CPU
4. 🤖 Agent> CPU 使用率: 12.5%...
================

🤖 You> clear
对话历史已清空

🤖 You> history
暂无对话历史
```

### 运行时切换

```bash
🤖 You> stream
流式输出已禁用

🤖 You> 你好
🤖 Agent> 你好！我是...

🤖 You> stream
流式输出已启用

🤖 You> 你好
🤖 Agent> 你好！我是...（逐字输出）
```

---

## 🔧 技术要点

### 1. strings.Builder 使用

```go
var fullContent strings.Builder
fullContent.WriteString(chunk)
return fullContent.String()
```

优势：
- 比字符串拼接高效
- 避免多次内存分配
- 线程安全（单 goroutine 使用）

### 2. 回调函数模式

```go
func ProcessStream(ctx, userInput, streamCallback func(string)) error {
    err := llmClient.ChatStream(ctx, messages, functions, func(chunk string) {
        if streamCallback != nil {
            streamCallback(chunk)  // 传递给上层
        }
    })
}
```

优点：
- 分层清晰
- 灵活性高
- 易于测试

### 3. 消息历史管理

```go
// 添加用户消息
a.messages = append(a.messages, llm.Message{
    Role:    "user",
    Content: userInput,
})

// 保存助手回复
a.messages = append(a.messages, llm.Message{
    Role:    "assistant",
    Content: fullContent.String(),
})
```

特点：
- 自动维护对话上下文
- 保留完整对话历史
- 支持多轮对话

---

## ✅ 验收标准

- [x] agent.ProcessStream() 实现
- [x] session.ProcessStream() 实现
- [x] CLI 根据 stream 配置选择输出模式
- [x] stream 命令切换流式输出
- [x] GetHistory() 返回原始历史
- [x] GetHistoryFormatted() 返回格式化历史
- [x] history 命令查看历史
- [x] clear 命令清空历史（已存在）
- [x] 实时 token 输出
- [x] 自动换行处理
- [x] 编译成功
- [x] 代码提交到 GitHub
- [x] 记录到 .dev_history

---

## 🎯 后续优化

### Phase 3.5: 用户体验增强

- [ ] 流式输出时支持停止（Ctrl+C）
- [ ] 历史记录持久化到文件
- [ ] 历史记录搜索功能
- [ ] 历史记录导出（JSON/Markdown）
- [ ] 多会话支持

### Phase 4: 性能优化

- [ ] 流式输出缓冲区优化
- [ ] 历史记录压缩（长期存储）
- [ ] 增量历史更新

---

## 🔗 参考资源

### 相关文档
- [OpenAI Streaming](https://platform.openai.com/docs/api-reference/streaming)
- [Go strings.Builder](https://pkg.go.dev/strings#Builder)

### 设计模式
- **Callback Pattern**: 分层处理流式数据
- **Builder Pattern**: strings.Builder 高效拼接
- **Strategy Pattern**: 流式/非流式切换

---

## 📝 Git 提交

**Commit**: `9dffd4b`
**Message**: 实现流式输出和对话历史功能

**Files Changed**:
- `internal/agent/agent.go` (修改)
- `internal/chat/session.go` (重构)
- `cmd/chat.go` (修改)

---

## 🎓 经验总结

### 成功点
1. **分层设计**: LLM → Agent → Session → CLI 清晰分层
2. **运行时切换**: stream 命令灵活切换输出模式
3. **双格式历史**: 原始和格式化两种格式满足不同需求
4. **性能优化**: strings.Builder 提升字符串处理效率

### 技术难点
1. **流式输出换行**: 需要判断最后一个字符是否换行符
2. **历史格式化**: 表情符号和角色映射
3. **模式切换**: 运行时动态选择处理方法

### 待改进
1. **流式停止**: 需要支持 Ctrl+C 停止当前输出
2. **历史持久化**: 当前历史只在内存中，重启丢失
3. **历史搜索**: 大量历史时需要搜索功能
4. **多会话**: 支持同时维护多个会话历史

---

**记录时间**: 2025-03-19 18:30
**记录人**: Claude Sonnet 4.6
