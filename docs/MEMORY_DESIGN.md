# Arch Linux AI Agent 记忆管理设计方案

基于论文 *"Memory in the Age of AI Agents: A Survey — Forms, Functions and Dynamics"* (arXiv:2512.13564)

---

## 📚 论文核心框架

### Forms-Functions-Dynamics 三维分类

**传统分类的不足**：论文指出传统的"长/短期记忆"二分法已不足以描述现代 Agent 记忆系统。

#### 1. Forms (形式) - 记忆存储在哪里

| 形态 | 描述 | 实现方式 | 适用场景 |
|------|------|----------|----------|
| **Token-level** | 直接存储在上下文窗口 | 系统提示、对话历史 | 短期、高频访问的记忆 |
| **Parametric** | 编码在模型参数中 | 微调、LoRA | 领域知识、操作模式 |
| **Latent** | 向量表示 | 向量数据库 + Embedding | 长期记忆、语义检索 |

#### 2. Functions (功能) - 记忆做什么

- **Formation**: 记忆如何形成（提取、压缩、索引）
- **Retrieval**: 如何检索记忆（相似度、时间、重要性）
- **Evolution**: 记忆如何演化（更新、遗忘、合并）

#### 3. Dynamics (动态) - 记忆如何变化

- 记忆生命周期管理
- 重要性衰减
- 冲突解决

---

## 🎯 Arch Linux Agent 记忆架构设计

### 整体架构图

```
┌─────────────────────────────────────────────────────────────┐
│                     AI Agent Memory System                   │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌───────────────┐  ┌──────────────┐  ┌──────────────────┐ │
│  │ Token-level   │  │   Latent     │  │  Parametric      │ │
│  │ Memory        │  │   Memory     │  │  Memory          │ │
│  │               │  │              │  │                  │ │
│  │ • 对话历史    │  │ • 向量数据库 │  │ • 系统知识       │ │
│  │ • 当前上下文  │  │ • 语义索引   │  │ • 操作模式       │ │
│  │ • 工作状态    │  │ • 长期记忆   │  │ (可选微调)       │ │
│  └───────────────┘  └──────────────┘  └──────────────────┘ │
│         │                    │                    │         │
│         └────────────────────┴────────────────────┘         │
│                            │                                │
│                   ┌────────▼────────┐                       │
│                   │ Memory Manager  │                       │
│                   │                 │                       │
│                   │ • Formation     │                       │
│                   │ • Retrieval     │                       │
│                   │ • Evolution     │                       │
│                   │ • Lifecycle     │                       │
│                   └────────────────┘                       │
└─────────────────────────────────────────────────────────────┘
```

---

## 🏗️ 详细设计方案

### 1. Token-level Memory (令牌级记忆)

#### 存储结构
```go
// internal/memory/token_memory.go

type TokenMemory struct {
    // 当前对话上下文
    ConversationHistory []Message

    // 当前会话状态
    SessionContext struct {
        CurrentTask     string
        PendingActions  []Action
        SystemState     SystemInfo
        UserPreferences map[string]string
    }

    // 工作记忆（临时存储）
    WorkingMemory struct {
        LastCommand    string
        LastOutput     string
        ErrorHistory   []Error
    }
}

type Message struct {
    Role      string  // user/assistant/system
    Content   string
    Timestamp time.Time
    Metadata  map[string]string
}
```

#### 配置参数
```yaml
memory:
  token:
    max_history: 50           # 最大保留消息数
    max_tokens: 8000          # 最大 token 数
    summary_threshold: 30     # 超过多少条消息触发摘要
```

#### 实现策略

**自动摘要机制**：
```go
// 当对话历史过长时，自动摘要旧消息
func (tm *TokenMemory) SummarizeOldMessages(llm *LLMClient) error {
    if len(tm.ConversationHistory) < tm.config.SummaryThreshold {
        return nil
    }

    // 保留最近 N 条消息
    recent := tm.ConversationHistory[-tm.config.KeepRecent:]
    old := tm.ConversationHistory[:len(tm.ConversationHistory)-tm.config.KeepRecent]

    // 使用 LLM 生成摘要
    summary, _ := llm.Summarize(old)

    // 将摘要作为系统消息插入
    tm.ConversationHistory = []Message{
        {Role: "system", Content: "[Previous conversation summary]\n" + summary},
    }
    tm.ConversationHistory = append(tm.ConversationHistory, recent...)

    return nil
}
```

---

### 2. Latent Memory (潜在记忆 / 向量记忆)

#### 存储选型

对于 Arch Linux Agent（本地 CLI 工具），推荐：

| 方案 | 优点 | 缺点 | 推荐度 |
|------|------|------|--------|
| **SQLite + pgvector** | 轻量、无需额外服务 | 性能中等 | ⭐⭐⭐⭐⭐ |
| **Chroma (本地)** | 纯 Python/Go，简单 | 需要额外进程 | ⭐⭐⭐⭐ |
| **Qdrant (本地)** | 性能好、功能丰富 | 较重 | ⭐⭐⭐ |
| **纯文件向量** | 最简单 | 检索效率低 | ⭐⭐ |

**推荐方案**: SQLite + pgvector（单文件、轻量、可靠）

#### 数据结构

```go
// internal/memory/latent_memory.go

type Memory struct {
    ID        string    `json:"id"`
    Type      MemoryType `json:"type"`
    Content   string    `json:"content"`
    Embedding []float32 `json:"embedding"`
    Metadata  Metadata  `json:"metadata"`
    CreatedAt time.Time `json:"created_at"`
    AccessedAt time.Time `json:"accessed_at"`
    AccessCount int     `json:"access_count"`
    Importance float64  `json:"importance"`  // 0-1
}

type MemoryType string

const (
    MemoryTypeSystem      MemoryType = "system"      // 系统知识
    MemoryTypeCommand     MemoryType = "command"     // 命令执行记录
    MemoryTypeProblem     MemoryType = "problem"     // 问题解决
    MemoryTypePreference  MemoryType = "preference"  // 用户偏好
    MemoryTypePattern     MemoryType = "pattern"     // 操作模式
)

type Metadata struct {
    Command     string            `json:"command,omitempty"`
    ExitCode    int               `json:"exit_code,omitempty"`
    Duration    time.Duration     `json:"duration,omitempty"`
    Success     bool              `json:"success,omitempty"`
    Tags        []string          `json:"tags,omitempty"`
    Context     map[string]string `json:"context,omitempty"`
}
```

#### Schema 设计

```sql
-- memories.sql
CREATE TABLE memories (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    content TEXT NOT NULL,
    embedding BLOB,  -- vector(1536)
    metadata_json TEXT,
    created_at TIMESTAMP,
    accessed_at TIMESTAMP,
    access_count INTEGER DEFAULT 0,
    importance REAL DEFAULT 0.5
);

-- 向量相似度索引
CREATE INDEX idx_embedding ON memories USING ivfflat (embedding vector_cosine_ops);

-- 类型索引
CREATE INDEX idx_type ON memories(type);

-- 时间索引
CREATE INDEX idx_created ON memories(created_at);
```

#### 记忆形成 (Formation)

```go
// 自动记忆提取
func (lm *LatentMemory) FormFromInteraction(
    ctx context.Context,
    userInput string,
    agentResponse string,
    toolCalls []ToolCall,
) error {

    memories := []Memory{}

    // 1. 提取命令执行记忆
    for _, call := range toolCalls {
        if call.Success {
            memories = append(memories, Memory{
                Type:    MemoryTypeCommand,
                Content: fmt.Sprintf("Command: %s → Success", call.Command),
                Metadata: Metadata{
                    Command:  call.Command,
                    ExitCode: call.ExitCode,
                    Success:  true,
                },
                Tags: []string{"success", call.ToolName},
            })
        }
    }

    // 2. 提取问题解决记忆
    if !strings.Contains(agentResponse, "成功") && len(toolCalls) > 0 {
        memories = append(memories, Memory{
            Type:    MemoryTypeProblem,
            Content: fmt.Sprintf("Problem: %s\nSolution: %s", userInput, agentResponse),
            Tags:    []string{"troubleshooting"},
        })
    }

    // 3. 提取系统知识
    sysKnowledge := lm.extractSystemKnowledge(toolCalls)
    memories = append(memories, sysKnowledge...)

    // 4. 生成 Embedding 并存储
    for _, mem := range memories {
        mem.Embedding = lm.embeddingClient.Embed(mem.Content)
        mem.CreatedAt = time.Now()
        lm.Store(mem)
    }

    return nil
}
```

#### 记忆检索 (Retrieval)

```go
// 混合检索策略
type RetrievalStrategy int

const (
    RetrieveBySimilarity  RetrievalStrategy = iota  // 语义相似度
    RetrieveByRecency                                // 时间最近
    RetrieveByImportance                             // 重要性
    RetrieveByType                                   // 类型过滤
    RetrieveHybrid                                  // 混合策略
)

func (lm *LatentMemory) Retrieve(
    ctx context.Context,
    query string,
    strategy RetrievalStrategy,
    limit int,
) ([]Memory, error) {

    queryEmbedding := lm.embeddingClient.Embed(query)

    switch strategy {
    case RetrieveBySimilarity:
        // 向量相似度检索
        return lm.vectorSearch(queryEmbedding, limit)

    case RetrieveByRecency:
        // 最近访问的记忆
        return lm.db.Where("accessed_at > ?", time.Now().Add(-7*24*time.Hour)).
            OrderBy("accessed_at DESC").Limit(limit).Find(&[]Memory{}).Error

    case RetrieveByImportance:
        // 高重要性记忆
        return lm.db.Where("importance > ?", 0.7).
            OrderBy("importance DESC").Limit(limit).Find(&[]Memory{}).Error

    case RetrieveHybrid:
        // 混合：相似度 * importance * recency_score
        return lm.hybridSearch(queryEmbedding, limit)
    }

    return nil, nil
}

// RAG 注入到 Token-level Memory
func (lm *LatentMemory) InjectToContext(
    ctx context.Context,
    query string,
    tokenMem *TokenMemory,
    maxCount int,
) error {

    // 检索相关记忆
    memories, _ := lm.Retrieve(ctx, query, RetrieveHybrid, maxCount)

    // 构建上下文
    contextStr := "[Relevant Memories]\n"
    for _, mem := range memories {
        contextStr += fmt.Sprintf("- %s: %s\n", mem.Type, mem.Content)
        // 更新访问统计
        mem.AccessCount++
        mem.AccessedAt = time.Now()
        lm.db.Save(&mem)
    }

    // 注入到对话历史
    tokenMem.ConversationHistory = append([]Message{
        {Role: "system", Content: contextStr},
    }, tokenMem.ConversationHistory...)

    return nil
}
```

#### 记忆演化 (Evolution)

```go
// 重要性衰减
func (lm *LatentMemory) DecayImportance() {
    lm.db.Model(&Memory{}).
        Where("accessed_at < ?", time.Now().Add(-30*24*time.Hour)).
        UpdateColumn("importance", gorm.Expr("importance * 0.9"))
}

// 记忆合并（去重）
func (lm *LatentMemory) MergeDuplicates() {
    // 找出相似记忆（余弦相似度 > 0.95）
    // 合并为一条，保留访问次数最多的
}

// 记忆遗忘
func (lm *LatentMemory) ForgetUnimportant() {
    // 删除重要性 < 0.2 且超过 90 天未访问的记忆
    lm.db.Where("importance < ? AND accessed_at < ?", 0.2, time.Now().Add(-90*24*time.Hour)).
        Delete(&Memory{})
}
```

---

### 3. Parametric Memory (参数记忆)

对于 Arch Linux Agent，Parametric Memory 有两种实现方式：

#### 方案 A: 领域知识 Prompt（推荐用于 v1）

```go
// 内部知识库，硬编码在 System Prompt 中
var ArchLinuxKnowledgebase = `
你是 Arch Linux 系统管理专家，具备以下领域知识：

[包管理]
- pacman: 官方包管理器
- yay: AUR 助手
- 常用命令: -Syu (升级), -Ss (搜索), -S (安装), -R (删除)

[服务管理]
- systemctl: systemd 服务控制
- 服务文件位置: /etc/systemd/system/
- 用户服务: ~/.config/systemd/user/

[系统监控]
- 系统信息: /proc 文件系统
- 日志: journalctl
- 进程: ps, top

[常见问题]
- 密钥问题: pacman-key --init && pacman-key --populate archlinux
- 依赖冲突: 检查 /var/log/pacman.log
...
`
```

#### 方案 B: 微调（可选，用于 v2+）

当有大量特定领域数据时，可以使用 LoRA 微调：

```python
# 微调数据集准备
{
    "instruction": "如何查看系统启动时间？",
    "input": "",
    "output": "使用 systemd-analyze 命令可以查看系统启动时间：\nsystemd-analyze\nsystemd-analyze blame  # 查看每个服务启动时间"
}

# 使用 LoRA 微调（需要大量训练数据）
```

---

## 🔄 记忆生命周期管理

### Lifecycle 状态机

```
┌─────────────┐
│  Formation  │  ← 创建记忆
└──────┬──────┘
       │
       ▼
┌─────────────┐
│    Active   │  ← 活跃使用期
└──────┬──────┘
       │
       ├─────────────────┐
       │                 │
       ▼                 ▼
┌─────────────┐   ┌─────────────┐
│  Dormant    │   │  Archived   │
│ (低频访问)  │   │  (长期存储) │
└──────┬──────┘   └─────────────┘
       │
       ▼
┌─────────────┐
│   Deleted   │  ← 遗忘/清理
└─────────────┘
```

### 定期维护任务

```go
// internal/memory/lifecycle.go

type LifecycleManager struct {
    latentMem *LatentMemory
    tokenMem  *TokenMemory
}

func (lm *LifecycleManager) RunMaintenance(ctx context.Context) {
    ticker := time.NewTicker(1 * time.Hour)
    for {
        select {
        case <-ticker.C:
            lm.maintenanceTasks()
        case <-ctx.Done():
            return
        }
    }
}

func (lm *LifecycleManager) maintenanceTasks() {
    // 1. Token Memory 摘要
    if lm.tokenMem.NeedsSummary() {
        lm.tokenMem.Summarize()
    }

    // 2. Latent Memory 重要性衰减
    lm.latentMem.DecayImportance()

    // 3. 合并重复记忆
    lm.latentMem.MergeDuplicates()

    // 4. 清理低价值记忆
    lm.latentMem.ForgetUnimportant()

    // 5. 更新记忆重要性（基于访问频率）
    lm.latentMem.RecalculateImportance()
}
```

---

## 🛠️ 技术选型

### Embedding 模型

| 模型 | 维度 | 性能 | 推荐场景 |
|------|------|------|----------|
| **text-embedding-3-small** (OpenAI) | 1536 | 高 | 通用 |
| **text-embedding-ada-002** | 1536 | 中 | 备选 |
| **bge-m3** (本地) | 1024 | 高 | 离线 |
| **all-MiniLM-L6-v2** (本地) | 384 | 中 | 轻量 |

**推荐**: `text-embedding-3-small`（性价比高）

### Go 库选型

```go
// go.mod
require (
    github.com/lib/pq           // pgvector 支持
    github.com/pgvector/pgvector-go
    github.com/tmc/langchaingo  // Embedding 支持
    gorm.io/gorm                // ORM
    gorm.io/driver/sqlite
)
```

### 本地向量库对比

| 库 | 支持语言 | 推荐度 |
|------|----------|--------|
| **pgvector** | SQL + Go | ⭐⭐⭐⭐⭐ |
| **chroma-go** | Go | ⭐⭐⭐⭐ |
| **qdrant-go** | Go | ⭐⭐⭐ |
| **weaviate-go** | Go | ⭐⭐⭐ |

---

## 📝 使用示例

### 完整对话流程

```go
// 1. 用户输入
userInput := "帮我检查一下为什么上次 nginx 启动失败"

// 2. Token Memory 检索相关历史
tokenMem := memory.NewTokenMemory()
latentMem := memory.NewLatentMemory()

// 3. 从 Latent Memory 检索相关记忆
latentMem.InjectToContext(ctx, userInput, tokenMem, 5)

// 4. 构建 Prompt（包含 Token + Latent）
prompt := buildPrompt(tokenMem)

// 5. LLM 决策
response := llm.Chat(prompt)

// 6. 执行工具调用
toolCalls := response.ToolCalls
results := executeTools(toolCalls)

// 7. 形成新记忆
latentMem.FormFromInteraction(ctx, userInput, response.Content, results)

// 8. 更新 Token Memory
tokenMem.Append(Message{
    Role:      "assistant",
    Content:   response.Content,
    Timestamp: time.Now(),
})

// 9. 定期维护
lifecycleManager.RunMaintenance(ctx)
```

---

## 🚀 实施计划

### Phase 1: 基础记忆 (Week 1-2)
- [x] 设计 Token Memory 结构
- [ ] 实现对话历史管理
- [ ] 实现自动摘要机制
- [ ] 实现 SQLite + pgvector

### Phase 2: Latent Memory (Week 3-4)
- [ ] 实现 Embedding 集成
- [ ] 实现记忆形成逻辑
- [ ] 实现向量检索
- [ ] 实现 RAG 注入

### Phase 3: 生命周期管理 (Week 5)
- [ ] 实现重要性计算
- [ ] 实现记忆衰减
- [ ] 实现定期维护任务
- [ ] 实现记忆合并

### Phase 4: 优化与评估 (Week 6)
- [ ] 性能优化
- [ ] 内存使用优化
- [ ] 准确率评估
- [ ] 文档完善

---

## 📊 评估指标

| 指标 | 描述 | 目标值 |
|------|------|--------|
| **检索准确率** | 相关记忆召回率 | > 80% |
| **响应延迟** | 记忆检索耗时 | < 100ms |
| **内存占用** | 向量数据库大小 | < 100MB |
| **记忆效用** | 成功使用的记忆占比 | > 60% |

---

## 🔗 参考资料

- [Memory in the Age of AI Agents: A Survey](https://arxiv.org/abs/2512.13564) - 原论文
- [知乎详细解析](https://zhuanlan.zhihu.com/p/1988419727622677787) - 中文解读
- [Agent Memory Paper List](https://github.com/Shichun-Liu/Agent-Memory-Paper-List) - 相关论文列表
- [pgvector Documentation](https://github.com/pgvector/pgvector) - 向量数据库
