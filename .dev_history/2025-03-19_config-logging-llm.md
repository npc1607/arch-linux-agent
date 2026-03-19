# 配置管理、日志系统和 LLM 集成

**完成时间**: 2025-03-19 16:30
**任务类型**: 核心基础设施实现
**状态**: ✅ 已完成

---

## 📋 任务概述

按照计划实现三大核心基础设施：
1. **配置管理系统** - 统一的配置加载和管理
2. **日志系统** - 结构化日志和日志轮转
3. **LLM 集成** - OpenAI 客户端和提示词管理

---

## 🏗️ 实现详情

### 1. 配置管理系统 (`internal/config/`)

#### 核心结构

```go
type Config struct {
    LLM      LLMConfig
    Security SecurityConfig
    Logging  LoggingConfig
    Commands CommandsConfig
    Memory   MemoryConfig
}
```

#### 功能特性

**多源配置**：
- YAML 配置文件 (`~/.config/arch-agent/config.yaml`)
- 环境变量 (`ARCH_AGENT_*`, `OPENAI_API_KEY`)
- 命令行参数 (`--api-key`, `--model`, 等)

**配置优先级**：命令行 > 环境变量 > 配置文件 > 默认值

**配置验证**：
- API Key 必填检查
- 模型名称验证
- 温度参数范围检查
- 日志级别验证
- 日志输出模式验证

**默认值设置**：
```go
- LLM:
  - model: "gpt-4o"
  - max_tokens: 4096
  - temperature: 0.7

- Security:
  - confirm_before_action: true
  - safe_mode: false
  - max_execution_time: 300s

- Logging:
  - level: "info"
  - output: "stdout"
  - max_size: 100MB
  - max_backups: 3
  - max_age: 7 days

- Memory:
  - enabled: true
  - token_max_history: 50
  - latent_max_memories: 10000
```

#### 配置文件示例

```yaml
llm:
  api_key: "sk-your-key-here"
  model: "gpt-4o"
  base_url: ""
  max_tokens: 4096
  temperature: 0.7

security:
  confirm_before_action: true
  safe_mode: false
  max_execution_time: 300

logging:
  level: "info"
  output: "stdout"
  file_path: "~/.local/state/arch-agent/agent.log"
  max_size: 100
  max_backups: 3
  max_age: 7
  compress: true

commands:
  pacman_cmd: "pacman"
  yay_cmd: "yay"
  sudo_cmd: "sudo"
  systemctl: "systemctl"

memory:
  enabled: true
  token_max_history: 50
  latent_max_memories: 10000
  auto_summary: true
  decay_interval: 24
```

---

### 2. 日志系统 (`pkg/logger/`)

#### 技术选型

- **核心库**: uber-go/zap（高性能结构化日志）
- **日志轮转**: lumberjack（自动压缩和清理）

#### 功能特性

**多种输出模式**：
- `stdout`: 仅控制台输出（开发）
- `file`: 仅文件输出（生产）
- `both`: 同时输出（调试）

**日志级别**：
- `debug`: 详细调试信息
- `info`: 一般信息（默认）
- `warn`: 警告信息
- `error`: 错误信息

**日志轮转**：
- 单文件最大 100MB（可配置）
- 保留最近 3 个备份文件
- 自动压缩旧日志
- 7 天后自动删除

#### 使用示例

```go
// 初始化日志系统
logger.Init(&logger.Config{
    Level:  "info",
    Output: "stdout",
})

// 记录日志
logger.Info("启动应用",
    logger.String("version", "1.0"),
    logger.Bool("debug", false),
)

logger.Debug("调试信息",
    logger.Int("count", 42),
)

logger.Error("发生错误",
    logger.Err(err),
    logger.String("context", "操作失败"),
)

// 格式化日志
logger.Infof("用户 %s 登录成功", username)

// 创建带字段的 logger
log := logger.With(
    logger.String("user_id", "123"),
)
log.Info("用户操作")
```

#### 日志格式

**控制台输出**（人类可读）：
```
2025-03-19T16:30:00+08:00	INFO	启动 Arch Linux AI Agent	{"version": "0.1.0", "mode": "chat"}
```

**文件输出**（JSON 格式）：
```json
{
  "time": "2025-03-19T16:30:00+08:00",
  "level": "INFO",
  "msg": "启动 Arch Linux AI Agent",
  "version": "0.1.0",
  "mode": "chat"
}
```

---

### 3. LLM 集成 (`internal/llm/`)

#### LLM 客户端 (`client.go`)

**功能**：
- OpenAI API 封装
- 流式/非流式对话
- Function Calling 支持
- Token 使用统计

**核心接口**：

```go
type Client struct {
    client *openai.Client
    config *config.LLMConfig
    model  string
}

// 非流式对话
func (c *Client) Chat(ctx, messages, functions) (*Response, error)

// 流式对话
func (c *Client) ChatStream(ctx, messages, functions, callback) error
```

**使用示例**：

```go
// 创建客户端
client, _ := llm.NewClient(&config.LLMConfig{
    APIKey: "sk-xxx",
    Model:  "gpt-4o",
})

// 非流式调用
resp, _ := client.Chat(ctx, []Message{
    {Role: "user", Content: "你好"},
}, nil)
fmt.Println(resp.Content)

// 流式调用
client.ChatStream(ctx, messages, nil, func(chunk string) {
    fmt.Print(chunk)  // 实时输出
})
```

#### 提示词构建器 (`prompts.go`)

**功能**：
- 动态系统信息收集
- 安全模式提示
- 工具描述注入
- 规则自定义

**系统信息收集**：
```go
func GetSystemInfo() string {
    // 操作系统、架构
    // 内核版本
    // 运行时间
    // 主机名
}
```

**使用示例**：

```go
// 基础提示词
prompt := llm.BuildDefaultSystemPrompt(false)

// 自定义提示词
builder := llm.NewSystemPromptBuilder().
    SetSafeMode(true).
    SetToolDesc("可用工具:\n- pacman\n- systemctl").
    SetSystemInfo(llm.GetSystemInfo()).
    SetRules("自定义规则...")
prompt := builder.Build()
```

**生成的提示词示例**：

```
你是一个 Arch Linux 系统管理助手，帮助用户完成系统管理和监控任务。

规则：
1. 只执行用户明确请求的操作
2. 涉及系统变更的操作需要告知用户风险
...

【安全模式已启用】
当前处于只读模式，只能执行查询类操作

当前系统信息：
- 操作系统: linux
- 架构: amd64
- 内核版本: 6.7.8-arch1-1
- 运行时间: 3天 5小时
- 主机名: archlinux
```

---

## 📝 代码统计

| 文件 | 行数 | 描述 |
|------|------|------|
| `internal/config/config.go` | 250+ | 配置管理核心 |
| `internal/config/convert.go` | 15 | 配置转换 |
| `pkg/logger/logger.go` | 210+ | 日志系统 |
| `internal/llm/client.go` | 260+ | LLM 客户端 |
| `internal/llm/prompts.go` | 170+ | 提示词构建 |
| `internal/chat/session.go` | 20+ | 集成更新 |
| `cmd/chat.go` | 100+ | 集成更新 |
| **总计** | **~1025** | - |

---

## 🎨 设计亮点

### 1. 配置三层覆盖

```
默认值 (config.go)
    ↓ 被配置文件覆盖
~/.config/arch-agent/config.yaml
    ↓ 被环境变量覆盖
ARCH_AGENT_*, OPENAI_API_KEY
    ↓ 被命令行参数覆盖
--api-key, --model, etc.
```

### 2. 日志动态切换

```bash
# 开发模式：控制台输出
arch-agent chat

# 生产模式：文件输出
# 在 config.yaml 中设置 output: "file"

# 调试模式：同时输出
# 在 config.yaml 中设置 output: "both"
```

### 3. 系统信息动态收集

```go
// 每次启动时收集最新系统信息
info := llm.GetSystemInfo()
// Kernel: 6.7.8-arch1-1
// Uptime: 3天 5小时
// Hostname: archlinux
```

### 4. 类型安全的配置转换

```go
// 自动转换配置类型
logger.Init(cfg.Logging.ToLoggerConfig())
```

---

## 🔄 集成流程

### 启动流程

```
用户启动
    │
    ▼
┌─────────────────┐
│ 加载配置文件     │ ← ~/.config/arch-agent/config.yaml
└────────┬────────┘
         │
    ┌────▼────┐
    │ 环境变量 │ ← ARCH_AGENT_*, OPENAI_API_KEY
    └────┬────┘
         │
    ┌────▼────┐
    │ CLI 参数 │ ← --api-key, --model, etc.
    └────┬────┘
         │
    ┌────▼────────┐
    │ 配置验证     │
    └────┬────────┘
         │
    ┌────▼────┐
    │ 初始化日志 │
    └────┬────┘
         │
    ┌────▼────┐
    │ 收集系统 │
    │ 信息     │
    └────┬────┘
         │
    ┌────▼────┐
    │ 构建提示 │
    │ 词       │
    └────┬────┘
         │
    ┌────▼────┐
    │ 创建 LLM │
    │ 客户端   │
    └────┬────┘
         │
    ┌────▼────┐
    │ 启动聊天 │
    │ 会话     │
    └─────────┘
```

---

## 🚀 使用示例

### 配置文件

```bash
# 创建配置文件
mkdir -p ~/.config/arch-agent
cat > ~/.config/arch-agent/config.yaml <<EOF
llm:
  api_key: "sk-your-key"
  model: "gpt-4o"

logging:
  level: "debug"
  output: "both"
  file_path: "~/.local/state/arch-agent/agent.log"
EOF
```

### 环境变量

```bash
# 设置 API Key
export OPENAI_API_KEY="sk-your-key"

# 设置模型
export ARCH_AGENT_LLM_MODEL="gpt-4-turbo"

# 设置日志级别
export ARCH_AGENT_LOGGING_LEVEL="debug"
```

### 命令行

```bash
# 使用配置文件
arch-agent chat

# 覆盖 API Key
arch-agent chat --api-key "sk-other-key"

# 启用安全模式
arch-agent chat --safe-mode

# 指定模型
arch-agent chat --model gpt-3.5-turbo

# 设置温度
arch-agent chat --temperature 0.3

# 组合使用
arch-agent chat --safe-mode --model gpt-4-turbo --temperature 0.5
```

---

## 📊 功能对比

| 功能 | 状态 | 说明 |
|------|------|------|
| 配置文件加载 | ✅ | YAML 格式 |
| 环境变量支持 | ✅ | ARCH_AGENT_* 前缀 |
| CLI 参数覆盖 | ✅ | 所有配置项 |
| 配置验证 | ✅ | 启动时验证 |
| 结构化日志 | ✅ | zap 实现 |
| 日志轮转 | ✅ | lumberjack |
| 多输出模式 | ✅ | stdout/file/both |
| LLM 客户端 | ✅ | OpenAI 封装 |
| 流式对话 | ✅ | 实时输出 |
| 提示词构建 | ✅ | 动态生成 |
| 系统信息收集 | ✅ | 自动获取 |

---

## 🎯 后续计划

### Phase 3: 工具系统
- [ ] 实现工具注册系统
- [ ] 实现 Function Calling
- [ ] 集成系统工具

### Phase 4: 记忆集成
- [ ] 集成 Token Memory
- [ ] 集成 Latent Memory
- [ ] 实现 RAG 检索

---

## ✅ 验收标准

- [x] 配置系统正常工作
- [x] 日志系统正常记录
- [x] LLM 客户端正常调用
- [x] 提示词正确生成
- [x] 系统信息正确收集
- [x] 配置验证生效
- [x] 日志轮转正常
- [x] 集成到 chat 命令
- [x] 代码提交到 GitHub
- [x] 记录到 .dev_history

---

## 🔗 参考资源

### 文档
- [Viper 文档](https://github.com/spf13/viper)
- [Zap 文档](https://github.com/uber-go/zap)
- [Lumberjack 文档](https://github.com/natefinch/lumberjack)
- [go-openai 文档](https://github.com/sashabaranov/go-openai)

### 相关
- 配置模板: `config.example.yaml`
- CLI 指南: `docs/CLI_GUIDE.md`

---

## 📝 Git 提交

**Commit**: `a1528f8`
**Message**: Implement configuration management, logging system and LLM integration

**Files Changed**:
- `internal/config/config.go` (新建)
- `internal/config/convert.go` (新建)
- `pkg/logger/logger.go` (新建)
- `internal/llm/client.go` (新建)
- `internal/llm/prompts.go` (新建)
- `internal/chat/session.go` (修改)
- `cmd/chat.go` (修改)
- `go.mod`, `go.sum` (更新)

---

## 🎓 经验总结

### 成功点
1. **分层配置**: 默认 → 文件 → 环境 → CLI，灵活且直观
2. **结构化日志**: JSON 格式便于日志分析工具处理
3. **提示词动态**: 系统信息实时获取，提示词更准确
4. **类型安全**: Go 类型系统避免配置错误

### 技术难点
1. **配置转换**: 需要在不同配置类型间转换
2. **日志初始化**: 需要在配置加载后才能初始化
3. **类型兼容**: go-openai 库的类型变更问题

### 待改进
1. **配置热更新**: 运行时重新加载配置
2. **日志采样**: 高频日志的采样机制
3. **错误重试**: LLM 调用的自动重试

---

**记录时间**: 2025-03-19 16:30
**记录人**: Claude Sonnet 4.6
