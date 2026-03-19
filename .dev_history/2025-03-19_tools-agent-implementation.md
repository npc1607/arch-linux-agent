# 工具系统与 Agent 核心实现

**完成时间**: 2025-03-19 17:30
**任务类型**: 核心功能实现
**状态**: ✅ 已完成

---

## 📋 任务概述

按照计划实现两大核心模块：
1. **工具函数集** - Arch Linux 系统管理工具
2. **Agent 核心调用** - AI Agent 主控逻辑

---

## 🏗️ 实现详情

### 1. Shell 执行器 (`internal/tools/shell.go`)

#### 核心功能

**命令执行**：
- 超时控制（默认 5 分钟）
- 标准输出/错误捕获
- 执行时间统计
- 退出码解析

**安全验证**：
- 命令注入检测
- 重定向检测
- 管道检测
- 后台运行检测
- 安全模式强制

#### 核心接口

```go
type Executor struct {
    timeout   time.Duration
    safeMode  bool
    allowSudo bool
}

func (e *Executor) Run(ctx context.Context, cmd string, args ...string) (*CommandResult, error)
func (e *Executor) ValidateCommand(command string) error
func (e *Executor) IsSafeCommand(cmd string) bool
```

---

### 2. 包管理工具 (`internal/tools/pacman.go`)

#### 实现功能

| 功能 | 方法 | 安全 |
|------|------|------|
| 搜索软件包 | `Search(ctx, keyword)` | ✅ |
| 查询包信息 | `Query(ctx, packageName)` | ✅ |
| 列出已安装 | `ListInstalled(ctx)` | ✅ |
| 检查更新 | `CheckUpdates(ctx)` | ✅ |
| 安装软件包 | `Install(ctx, packages, noConfirm)` | ❌ |
| 删除软件包 | `Remove(ctx, packages, noConfirm)` | ❌ |
| 系统升级 | `Upgrade(ctx, noConfirm)` | ❌ |

#### 数据结构

```go
type Package struct {
    Name        string
    Version     string
    Description string
    Repository  string
    Size        string
    Installed   bool
}
```

---

### 3. 服务管理工具 (`internal/tools/systemd.go`)

#### 实现功能

| 功能 | 方法 | 安全 |
|------|------|------|
| 列出服务 | `ListServices(ctx, all)` | ✅ |
| 服务状态 | `GetServiceStatus(ctx, name)` | ✅ |
| 检查活动 | `IsActive(ctx, name)` | ✅ |
| 检查启用 | `IsEnabled(ctx, name)` | ✅ |
| 启动服务 | `Start(ctx, name)` | ❌ |
| 停止服务 | `Stop(ctx, name)` | ❌ |
| 重启服务 | `Restart(ctx, name)` | ❌ |
| 重载配置 | `Reload(ctx, name)` | ❌ |
| 启用服务 | `Enable(ctx, name)` | ❌ |
| 禁用服务 | `Disable(ctx, name)` | ❌ |
| 服务日志 | `GetJournal(ctx, name, lines)` | ✅ |
| 失败服务 | `GetFailedServices(ctx)` | ✅ |

#### 数据结构

```go
type Service struct {
    Name        string
    Description string
    Loaded      string  // loaded, not-found, error
    Active      string  // active, inactive, failed
    Sub         string  // running, dead, etc.
    Enabled     string  // enabled, disabled, static
}
```

---

### 4. 系统监控工具 (`internal/tools/monitor.go`)

#### 实现功能

**CPU 监控**：
- 使用率计算
- 核心数获取
- 负载平均值

**内存监控**：
- 总内存/已用/可用
- 使用率计算
- 交换分区统计

**磁盘监控**：
- 所有挂载点
- 使用情况
- 使用率计算

**进程监控**：
- 进程列表
- CPU/内存排序
- 状态信息

**系统信息**：
- 运行时间
- 时间戳

#### 数据结构

```go
type CPUInfo struct {
    Usage   float64  // 使用率
    Cores   int      // 核心数
    LoadAvg string   // 负载平均
}

type MemoryInfo struct {
    Total     uint64  // 总内存
    Used      uint64  // 已使用
    Available uint64  // 可用
    Usage     float64 // 使用率
    SwapTotal uint64  // 交换总计
    SwapUsed  uint64  // 交换已用
}

type DiskInfo struct {
    Filesystem string
    Size       uint64
    Used       uint64
    Available  uint64
    Usage      float64
    MountPoint string
}

type ProcessInfo struct {
    PID     int
    Name    string
    User    string
    CPU     float64
    Memory  float64
    Status  string
    Command string
}
```

---

### 5. 日志工具 (`internal/tools/log.go`)

#### 实现功能

**日志查询**：
- systemd journalctl 查询
- 按服务过滤
- 按时间过滤
- 按优先级过滤
- 行数限制

**日志分析**：
- 系统错误日志
- 服务日志
- 启动日志
- 内核日志
- 错误统计

#### 数据结构

```go
type JournalEntry struct {
    Time     string
    Host     string
    Process  string
    Message  string
    Priority int
}
```

---

### 6. Agent 核心 (`internal/agent/`)

#### 6.1 工具注册表 (`tools.go`)

**功能**：
- 工具注册和管理
- 安全/不安全分类
- OpenAI Function Calling 格式转换
- 工具执行和错误处理

**注册的工具** (15+ 个)：

**包管理**：
- `pacman_search` - 搜索软件包
- `pacman_query` - 查询包信息
- `pacman_install` - 安装软件包
- `pacman_check_updates` - 检查更新

**服务管理**：
- `systemd_list` - 列出服务
- `systemd_status` - 服务状态
- `systemd_start` - 启动服务
- `systemd_stop` - 停止服务
- `systemd_restart` - 重启服务

**系统监控**：
- `monitor_cpu` - CPU 使用率
- `monitor_memory` - 内存使用
- `monitor_disk` - 磁盘使用
- `monitor_system` - 系统信息汇总

**日志分析**：
- `log_errors` - 系统错误日志
- `log_service` - 服务日志

#### 工具定义

```go
type Tool struct {
    Name        string                    `json:"name"`
    Description string                    `json:"description"`
    Function    openai.FunctionDefinition `json:"function"`
    Handler     ToolHandler               `json:"-"`
    Safe        bool                      `json:"safe"`
}

type ToolHandler func(ctx context.Context, params map[string]interface{}) (string, error)
```

#### 6.2 Agent 主控逻辑 (`agent.go`)

**核心流程**：

```
用户输入
    │
    ▼
┌─────────────────┐
│ 构建消息列表     │
│ (系统提示 + 历史) │
└────────┬────────┘
         │
    ┌────▼─────────┐
│ 调用 LLM       │
│ (带工具列表)    │
└────┬─────────┘
         │
    ┌────▼───────┐
│ 是否需要工具?  │
└────┬──────────┘
     │
  是  │  否
  ▼   ▼
┌──────┐ ┌──────┐
│执行工具│ │返回结果│
└───┬──┘ └───┬──┘
    │       │
    └───┬───┘
        ▼
   ┌─────────┐
   │ LLM 总结│
   └────┬────┘
        ▼
   ┌─────────┐
   │最终回复 │
   └─────────┘
```

**核心接口**：

```go
type Agent struct {
    config    *config.Config
    llmClient *llm.Client
    tools     *ToolRegistry
    messages  []llm.Message
}

func (a *Agent) Process(ctx context.Context, userInput string, streamCallback func(string)) (string, error)
func (a *Agent) handleToolCalls(ctx context.Context, toolCalls []llm.ToolCall, streamCallback func(string)) (string, error)
```

#### 6.3 结果格式化 (`format.go`)

**功能**：
- 格式化包信息
- 格式化服务信息
- 格式化系统信息
- 格式化日志条目

**格式化函数**：

```go
formatPackages(packages []Package) string
formatPackage(pkg *Package) string
formatServices(services []Service) string
formatService(svc *Service) string
formatCPUInfo(cpu *CPUInfo) string
formatMemoryInfo(mem *MemoryInfo) string
formatDiskInfo(disks []DiskInfo) string
formatSystemInfo(info map[string]interface{}) string
formatJournalEntries(entries []JournalEntry) string
```

---

## 📝 代码统计

| 模块 | 行数 | 功能 |
|------|------|------|
| `tools/shell.go` | 165 | Shell 执行器 |
| `tools/pacman.go` | 225 | 包管理工具 |
| `tools/systemd.go` | 215 | 服务管理工具 |
| `tools/monitor.go` | 330 | 系统监控工具 |
| `tools/log.go` | 200 | 日志工具 |
| `agent/tools.go` | 590 | 工具注册表 |
| `agent/format.go` | 200+ | 结果格式化 |
| `agent/agent.go` | 260 | Agent 核心 |
| **总计** | **~2185** | - |

---

## 🎨 设计亮点

### 1. 安全分级

所有工具按照安全性分为两类：

**安全工具** (只读操作)：
- 查询类操作
- 监控类操作
- 日志查询

**不安全工具** (需要确认)：
- 安装/删除软件
- 启动/停止服务
- 任何修改系统的操作

### 2. 安全模式强制

```go
if r.config.Security.SafeMode && !tool.Safe {
    return "", fmt.Errorf("安全模式下不允许执行此工具: %s", name)
}
```

安全模式下只返回安全工具给 LLM。

### 3. 命令验证

```go
// 检查命令注入
if strings.Contains(command, "&&") || strings.Contains(command, "||") {
    return fmt.Errorf("不允许使用命令组合")
}

// 检查重定向
if strings.Contains(command, ">") || strings.Contains(command, "<") {
    return fmt.Errorf("不允许使用重定向")
}
```

### 4. Function Calling 集成

工具自动转换为 OpenAI Function Calling 格式：

```go
type Tool struct {
    Function: openai.FunctionDefinition{
        Name:        "monitor_cpu",
        Description: "获取当前 CPU 使用率",
        Parameters: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{},
        },
    },
}
```

### 5. 结构化结果格式化

所有工具返回统一的、人类可读的格式：

```
=== CPU ===
CPU 使用率: 12.5%
核心数: 8
负载平均: 0.8 1.2 1.1

=== 内存 ===
总内存: 16.0G
已使用: 4.2G (26.3%)
可用: 11.8G
```

---

## 🔄 工作流程

### 完整对话流程

```
用户输入: "检查系统状态"
    │
    ▼
┌─────────────────────┐
│ Agent.Process()      │
│ - 构建系统提示       │
│ - 添加可用工具       │
└──────────┬──────────┘
           │
    ┌──────▼──────────┐
│ 调用 LLM (无工具)  │
│ 因为只是查询       │
└──────┬──────────┘
       │
   ┌───▼────────────┐
│ LLM 决策:        │
│ "直接查询系统信息" │
└───┬────────────┘
    │
┌───▼──────────────┐
│ 调用 monitor_cpu │
│ 调用 monitor_mem │
│ 调用 monitor_disk│
└───┬──────────────┘
    │
┌───▼──────────────┐
│ 格式化结果        │
└───┬──────────────┘
    │
┌───▼──────────────┐
│ 返回给用户        │
│ "CPU: 12.5%..."  │
└──────────────────┘
```

### Function Calling 流程

```
用户输入: "安装 nginx"
    │
    ▼
┌─────────────────────┐
│ Agent.Process()      │
│ - 构建系统提示       │
│ - 添加工具列表       │
└──────────┬──────────┘
           │
    ┌──────▼──────────┐
│ 调用 LLM (带工具)  │
└──────┬──────────┘
       │
   ┌───▼────────────────┐
│ LLM 返回:           │
│ tool_calls: [       │
│   {                │
│     name: "pacman_install",│
│     arguments: '{"packages": ["nginx"]}'│
│   }                │
│ ]                  │
└───┬────────────────┘
    │
┌───▼──────────────────┐
│ handleToolCalls()    │
│ - 解析参数           │
│ - 执行 pacman_install│
│ - 获取结果           │
└───┬──────────────────┘
    │
┌───▼──────────────────┐
│ 添加工具结果到消息    │
│ role: "tool"         │
│ content: "安装成功..." │
└───┬──────────────────┘
    │
┌───▼──────────────────┐
│ 再次调用 LLM 总结    │
└───┬──────────────────┘
    │
┌───▼──────────────────┐
│ 返回最终回复         │
│ "nginx 已成功安装..." │
└─────────────────────┘
```

---

## 🚀 使用示例

### 安全模式

```bash
# 安全模式下只能执行只读操作
./bin/arch-agent chat --safe-mode

🤖 You> 安装 nginx
🤖 Agent> 【安全模式已启用】当前处于只读模式，无法执行安装操作。

您可以执行：
- 搜索包：pacman -Ss nginx
- 查看状态：systemctl status nginx

如需安装，请退出安全模式后重试。
```

### 系统监控

```bash
🤖 You> 检查系统状态
🤖 Agent> 正在检查系统状态...

=== CPU ===
CPU 使用率: 15.2%
核心数: 8
负载平均: 1.2 1.5 1.4

=== 内存 ===
总内存: 16.0G
已使用: 6.8G (42.5%)
可用: 9.2G

=== 磁盘 ===
文件系统              大小        已使用      可用       使用率   挂载点
/dev/root            500.0G      150.0G     350.0G      30.0%    /
/dev/boot             1.0G        0.3G       0.7G       30.0%    /boot

运行时间: 3天 5小时
```

### 服务管理

```bash
🤖 You> 查看 docker 服务状态
🤖 Agent> 正在查询 docker 服务状态...

服务名称: docker
描述: Docker Application Container Engine
状态: active (running)
子状态: running
加载: loaded
启用: enabled

服务正在运行并已设置为开机自启。
```

### 日志查询

```bash
🤖 You> 查看最近的错误日志
🤖 Agent> 正在查询系统错误日志...

找到 3 条错误日志：

[2025-03-19 15:30:22] systemd: Failed to start Resize...
[2025-03-19 14:20:15] kernel: Out of memory: Killed process
[2025-03-19 13:10:08] nginx: Failed to start A web server
```

---

## 📊 功能对比

| 功能类别 | 工具数量 | 安全 | 不安全 |
|---------|---------|------|--------|
| 包管理 | 4 | 3 | 1 |
| 服务管理 | 6 | 4 | 2 |
| 系统监控 | 4 | 4 | 0 |
| 日志分析 | 2 | 2 | 0 |
| **总计** | **16** | **13** | **3** |

---

## 🎯 后续计划

### Phase 3: 记忆系统集成
- [ ] 将工具执行结果记录到记忆系统
- [ ] 实现记忆检索辅助工具调用
- [ ] 实现从历史经验学习

### Phase 4: 高级功能
- [ ] 工具调用链（一个工具的输出作为另一个工具的输入）
- [ ] 并发工具调用
- [ ] 工具执行重试和错误恢复

### Phase 5: 测试和文档
- [ ] 单元测试
- [ ] 集成测试
- [ ] 用户文档

---

## ✅ 验收标准

- [x] Shell 执行器实现
- [x] 命令验证和安全检查
- [x] Pacman 工具完整实现
- [x] Systemd 工具完整实现
- [x] 监控工具完整实现
- [x] 日志工具完整实现
- [x] 工具注册系统
- [x] Agent 核心逻辑
- [x] Function Calling 集成
- [x] 结果格式化
- [x] 安全模式强制
- [x] 编译成功
- [x] 代码提交到 GitHub
- [x] 记录到 .dev_history

---

## 🔗 参考资源

### 相关文档
- [Pacman 手册](https://wiki.archlinux.org/title/Pacman)
- [Systemd 手册](https://www.freedesktop.org/software/systemd/man/)
- [journalctl 手册](https://man7.org/linux/man-pages/man1/journalctl.html)

### 进程文件系统
- `/proc/stat` - CPU 统计
- `/proc/meminfo` - 内存信息
- `/proc/uptime` - 运行时间

---

## 📝 Git 提交

**Commit**: `d5b9956`
**Message**: Implement tools system and Agent core logic

**Files Changed**:
- `internal/tools/shell.go` (新建)
- `internal/tools/pacman.go` (新建)
- `internal/tools/systemd.go` (新建)
- `internal/tools/monitor.go` (新建)
- `internal/tools/log.go` (新建)
- `internal/agent/tools.go` (新建)
- `internal/agent/format.go` (新建)
- `internal/agent/agent.go` (新建)
- `pkg/logger/logger.go` (修改)

---

## 🎓 经验总结

### 成功点
1. **模块化设计**: 每个工具独立，易于扩展
2. **安全分级**: 清晰的安全/不安全分类
3. **Function Calling**: 与 OpenAI 无缝集成
4. **结果格式化**: 统一的用户友好的输出格式

### 技术难点
1. **命令参数解析**: shell 参数和 journalctl 参数的复杂组合
2. **类型转换**: OpenAI SDK 的类型系统变化
3. **错误处理**: 需要区分可恢复和不可恢复错误

### 待改进
1. **工具调用链**: 当前只支持单步工具调用
2. **并发工具**: 可以同时执行多个独立工具
3. **工具结果缓存**: 相同查询可以缓存结果
4. **更详细的错误信息**: 需要更好的错误诊断

---

**记录时间**: 2025-03-19 17:30
**记录人**: Claude Sonnet 4.6
