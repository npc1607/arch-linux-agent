# Agent 懒加载工具使用指南

## 快速开始

### 方式 1: 传统模式（预加载所有工具）

```go
import "github.com/npc1607/arch-linux-agent/internal/agent"

// 创建传统 Agent，所有工具在启动时预加载
agent, err := agent.NewAgent(cfg)
if err != nil {
    log.Fatal(err)
}

// 处理用户消息
response, err := agent.Process(ctx, "帮我安装一个软件包", nil)
```

### 方式 2: 懒加载模式（推荐）

```go
import "github.com/npc1607/arch-linux-agent/internal/agent"

// 创建支持懒加载的 Agent
// enableLazy=true 启用懒加载，工具按需加载
agent, err := agent.NewAgentWithLazyLoading(cfg, true)
if err != nil {
    log.Fatal(err)
}

// 处理用户消息
// 系统会自动检测消息中的关键词，按需加载相关工具
response, err := agent.Process(ctx, "帮我安装一个软件包", nil)
```

## 两种模式对比

| 特性 | 传统模式 | 懒加载模式 |
|------|---------|-----------|
| 启动时间 | 较慢（预加载所有工具） | 快速（只加载核心工具） |
| Token 消耗 | 高（~150k tokens） | 低（~20k tokens + 按需） |
| 响应速度 | 首次响应快 | 首次使用某工具组需加载 |
| 适用场景 | 工具数量少 | 工具数量多或不确定 |

## 懒加载工作流程

```
用户消息 → 检测关键词 → 提示加载 → 用户确认 → 加载工具组 → 执行操作
```

### 示例流程

1. **用户发送消息**: "帮我安装一个软件包"
2. **系统检测关键词**: 检测到 "安装" → 匹配 pacman 工具组
3. **自动加载**: 自动确认并加载 pacman 工具组（4个工具）
4. **执行操作**: AI 使用 pacman_install 工具完成操作

## 预配置的工具组

### 1. pacman - 包管理
**关键词**: `包`, `软件`, `安装`, `pacman`, `package`, `install`, `搜索`, `卸载`

**工具**:
- pacman_search - 搜索软件包
- pacman_query - 查询已安装的软件包信息
- pacman_install - 安装软件包
- pacman_check_updates - 检查系统更新

### 2. systemd - 服务管理
**关键词**: `服务`, `systemd`, `service`, `启动`, `停止`, `重启`, `状态`

**工具**:
- systemd_list - 列出系统服务
- systemd_status - 查看服务状态
- systemd_start - 启动服务
- systemd_stop - 停止服务
- systemd_restart - 重启服务

### 3. monitor - 系统监控
**关键词**: `监控`, `CPU`, `内存`, `磁盘`, `系统`, `monitor`, `system`, `资源`

**工具**:
- monitor_cpu - 获取 CPU 使用率
- monitor_memory - 获取内存使用情况
- monitor_disk - 获取磁盘使用情况
- monitor_system - 获取系统信息汇总

### 4. log - 日志管理
**关键词**: `日志`, `log`, `journal`, `错误`, `error`, `调试`

**工具**:
- log_errors - 查询系统错误日志
- log_service - 查询服务日志

### 5. weather - 天气查询
**关键词**: `天气`, `weather`, `温度`, `气温`, `预报`

**工具**:
- weather_get - 获取天气信息

## 查看工具加载状态

```go
// 获取工具组状态
status := agent.GetLazyRegistryStatus()
fmt.Println(agent.FormatLazyToolsStatus(status))
```

## 配置选项

在 `config.yaml` 中配置：

```yaml
agent:
  lazy_loading:
    enabled: true        # 是否启用懒加载
    auto_load: false     # 是否自动加载（false=需要确认）
    max_tools_per_group: 50  # 每组最大工具数
```

## 性能对比

### 传统模式
- 启动时间: ~2-3秒
- 初始 Token: ~150,000
- 适用: 工具数量少，需要快速响应

### 懒加载模式
- 启动时间: < 1秒
- 初始 Token: ~20,000
- 按需加载: 每组 ~5,000-10,000 tokens
- 适用: 工具数量多，启动速度优先

## 迁移指南

### 从传统模式迁移到懒加载模式

只需修改 Agent 创建方式：

```go
// 旧代码
agent, err := agent.NewAgent(cfg)

// 新代码
agent, err := agent.NewAgentWithLazyLoading(cfg, true)
```

其他 API 保持不变，无需修改业务代码。

## 最佳实践

1. **生产环境**: 推荐使用懒加载模式
2. **开发/调试**: 可使用传统模式，方便调试
3. **工具数量少**: < 10个工具，两种模式差异不大
4. **工具数量多**: > 20个工具，强烈推荐懒加载

## 注意事项

- 懒加载模式下，首次使用某个工具组会有轻微延迟（加载时间）
- 工具组一旦加载，在整个会话中保持可用
- 系统会自动检测关键词并加载，无需手动干预
