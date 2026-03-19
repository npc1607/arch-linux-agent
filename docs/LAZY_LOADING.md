# MCP 工具懒加载系统

基于 Claude Code 的 MCP Tool Search 功能实现，本系统提供按需加载工具的能力，大幅减少 token 消耗和启动时间。

## 🎯 核心特性

### 1. 按需加载 (On-Demand Loading)
- ✅ 只在需要时才加载工具定义
- ✅ 不会在会话开始时预加载所有工具
- ✅ 支持 90%+ 的 token 节省

### 2. 关键词触发 (Keyword-Triggered)
- ✅ 智能检测用户消息中的关键词
- ✅ 自动提示加载相关工具组
- ✅ 支持多语言关键词配置

### 3. 用户确认机制 (User Confirmation)
- ✅ 加载前请求用户确认
- ✅ 明确告知工具用途和影响
- ✅ 安全可控的工具加载流程

### 4. 会话持久化 (Session Persistence)
- ✅ 加载的工具在整个会话保持可用
- ✅ 避免重复加载
- ✅ 状态实时可查

## 📋 系统架构

```
┌─────────────────────────────────────────────────────────────┐
│                    LazyToolRegistry                          │
│                                                              │
│  ┌──────────────────┐      ┌─────────────────────────┐    │
│  │  ToolRegistry    │      │  LazyToolGroup[]        │    │
│  │  (基础工具)       │      │  (懒加载工具组)          │    │
│  │                  │      │                         │    │
│  │  - pacman        │      │  - github (未加载)      │    │
│  │  - systemd       │      │  - notion (未加载)      │    │
│  │  - monitor       │      │  - sentry (未加载)      │    │
│  │  - weather       │      │  - slack (未加载)       │    │
│  └──────────────────┘      └─────────────────────────┘    │
│                                      │                      │
│                              1. 关键词检测                   │
│                              2. 用户确认                     │
│                              3. 按需加载                     │
│                              4. 注册到 ToolRegistry         │
└─────────────────────────────────────────────────────────────┘
```

## 🚀 快速开始

### 基础用法

```go
package main

import (
    "context"
    "github.com/npc1607/arch-linux-agent/internal/agent"
)

func main() {
    ctx := context.Background()

    // 1. 创建基础工具注册表
    baseRegistry := agent.NewToolRegistry(config)

    // 2. 创建懒加载注册表
    lazyRegistry := agent.NewLazyToolRegistry(baseRegistry)

    // 3. 设置 MCP 工具组
    agent.SetupMockMCPServers(lazyRegistry)

    // 4. 获取工具摘要（轻量级）
    toolsSummary := lazyRegistry.GetToolSummary()

    // 5. 检查用户消息并自动加载
    userMessage := "帮我查一下 GitHub 上的代码"
    loadedGroups, err := lazyRegistry.CheckAndLoad(ctx, userMessage)

    // 6. 处理加载确认
    if agent.IsToolLoadPending(err) {
        groupName, _ := agent.GetGroupNameFromError(err)
        // 提示用户确认
        // 用户确认后加载
        lazyRegistry.LoadGroup(ctx, groupName, true)
    }

    // 7. 使用工具
    result, err := baseRegistry.Execute(ctx, "github_search_repositories", params)
}
```

## 📊 预配置的 MCP 工具组

### GitHub (development)
**关键词**: `github`, `repo`, `仓库`, `代码`, `commit`, `pr`, `issue`

**工具**:
- `github_search_repositories` - 搜索 GitHub 仓库
- `github_get_file` - 获取文件内容
- `github_list_issues` - 列出 Issues

### Notion (documentation)
**关键词**: `notion`, `文档`, `wiki`, `笔记`, `知识库`

**工具**:
- `notion_search_pages` - 搜索页面
- `notion_get_page` - 获取页面内容
- `notion_create_page` - 创建页面

### Sentry (debugging)
**关键词**: `sentry`, `错误`, `bug`, `error`, `调试`, `debug`, `崩溃`

**工具**:
- `sentry_list_errors` - 列出错误
- `sentry_get_error_details` - 获取错误详情
- `sentry_create_issue` - 从错误创建 Issue

### Slack (communication)
**关键词**: `slack`, `消息`, `通知`, `channel`, `频道`

**工具**:
- `slack_send_message` - 发送消息
- `slack_list_channels` - 列出频道

## ⚙️ 配置选项

### LazyLoadConfig

```go
type LazyLoadConfig struct {
    Enabled           bool   // 是否启用懒加载
    AutoLoad          bool   // 是否自动加载（无需确认）
    MaxToolsPerGroup  int    // 每组最大工具数
}
```

### LazyToolConfig

```go
type LazyToolConfig struct {
    Enabled         bool     // 是否启用此工具组
    TriggerKeywords []string // 触发关键词列表
    Description     string   // 工具组描述
    Category        string   // 分类标识
}
```

## 💡 使用场景

### 场景 1: 关键词自动触发

```go
// 用户消息
userMessage := "我在 GitHub 上有个项目，想搜索一些代码"

// 系统自动检测
loadedGroups, err := lazyRegistry.CheckAndLoad(ctx, userMessage)

// 检测到 "github" 关键词
// → 提示用户确认加载 GitHub 工具组
// → 用户确认后加载 3 个 GitHub 工具
```

### 场景 2: 手动加载工具组

```go
// 直接加载指定的工具组
tools, err := lazyRegistry.LoadGroup(ctx, "github", true)
// 参数: groupName, autoConfirm
```

### 场景 3: 查看工具状态

```go
// 获取所有工具组状态
status := lazyRegistry.GetGroupStatus()

// 格式化输出
formatted := agent.FormatLazyToolsStatus(status)
fmt.Println(formatted)
```

## 📈 性能对比

### 传统方式（全部预加载）

| 指标 | 数值 |
|------|------|
| 工具总数 | 142 |
| Token 消耗 | ~150,000 |
| 启动时间 | ~2-3 秒 |
| 实际使用 | ~6 个工具 (~6,600 tokens) |
| **浪费率** | **95%** |

### 懒加载方式（按需加载）

| 指标 | 数值 |
|------|------|
| 初始工具 | 基础工具 (~20,000 tokens) |
| Token 消耗 | ~20,000 + 按需加载 |
| 启动时间 | < 1 秒 |
| 按需加载 | 仅加载需要的工具组 |
| **节省率** | **90%+** |

## 🔧 自定义工具组

### 添加自定义 MCP 工具组

```go
// 1. 定义工具加载器
func loadMyCustomTools(ctx context.Context) ([]*Tool, error) {
    return []*Tool{
        {
            Name:        "my_tool",
            Description: "我的自定义工具",
            Function:    openai.FunctionDefinition{...},
            Handler:     myToolHandler,
            Safe:        true,
        },
    }, nil
}

// 2. 注册工具组
lazyRegistry.RegisterLazyGroup("my_custom", LazyToolConfig{
    Enabled:        true,
    TriggerKeywords: []string{"custom", "自定义"},
    Description:     "我的自定义工具组",
    Category:        "custom",
}, loadMyCustomTools)
```

## 📝 API 参考

### LazyToolRegistry 方法

| 方法 | 描述 |
|------|------|
| `RegisterLazyGroup(name, config, loader)` | 注册懒加载工具组 |
| `GetToolSummary()` | 获取工具摘要（不加载） |
| `LoadGroup(ctx, name, confirm)` | 加载指定工具组 |
| `CheckAndLoad(ctx, message)` | 检查消息并自动加载 |
| `GetLoadedTools()` | 获取已加载的工具 |
| `GetGroupStatus()` | 获取工具组状态 |
| `IsGroupLoaded(name)` | 检查工具组是否已加载 |

### 辅助函数

| 函数 | 描述 |
|------|------|
| `IsToolLoadPending(err)` | 检查是否是待加载错误 |
| `GetGroupNameFromError(err)` | 从错误获取工具组名 |
| `FormatLazyToolsStatus(status)` | 格式化工具组状态 |

## 🎨 示例输出

```
# 懒加载工具组状态

## ❌ github
**分类**: development
**描述**: GitHub 仓库和代码管理工具
**触发关键词**: github, repo, 仓库, 代码, commit, pr, issue
**状态**: 未加载（需要时自动加载）

## ✅ notion
**分类**: documentation
**描述**: Notion 文档和知识管理工具
**触发关键词**: notion, 文档, wiki, 笔记, 知识库
**工具数量**: 3

## ❌ sentry
**分类**: debugging
**描述**: Sentry 错误监控和调试工具
**触发关键词**: sentry, 错误, bug, error, 调试, debug, 崩溃
**状态**: 未加载（需要时自动加载）
```

## 🚦 运行示例

```bash
# 运行懒加载示例
go run ./internal/agent/lazy_example.go
```

## 📚 参考资料

- [Claude Code MCP Tool Search](https://medium.com/the-context-layer/claude-code-just-fixed-its-biggest-scaling-problem-with-mcp-tool-search-3aa1aebcd824)
- [GitHub Feature Request: Lazy-Loading MCP Tools](https://github.com/anthropics/claude-code/issues/16826)
- [Model Context Protocol (MCP)](https://modelcontextprotocol.io/)

## 🤝 贡献

欢迎提交 Issue 和 Pull Request 来改进这个懒加载系统！

## 📄 许可证

MIT License
