# Arch Linux AI Agent 项目计划

## 项目概述

开发一个基于 Go 语言的 AI Agent，通过自然语言交互来管理和监控 Arch Linux 系统。使用 OpenAI GPT 作为决策引擎，以 CLI 方式提供智能系统管理能力。

### 核心功能
- **系统管理**：pacman/yay 包管理、systemd 服务管理、用户管理
- **系统监控**：CPU/内存/磁盘监控、日志分析、进程监控
- **智能助手**：自然语言执行命令、故障排查、系统建议

### 技术栈
- **语言**：Go 1.23+
- **LLM**：OpenAI GPT API
- **CLI 框架**：cobra / promptui
- **系统交互**：exec 命令执行、D-Bus（systemd）
- **安全**：命令白名单、用户级别权限

---

## 项目目录结构

```
arch-linux-agent/
├── cmd/
│   └── root.go              # CLI 入口和根命令
├── internal/
│   ├── agent/              # Agent 核心逻辑
│   │   ├── agent.go        # Agent 主控逻辑
│   │   ├── planner.go      # 任务规划
│   │   └── executor.go     # 命令执行
│   ├── tools/              # 工具函数集
│   │   ├── pacman.go       # 包管理工具
│   │   ├── systemd.go      # 服务管理工具
│   │   ├── monitor.go      # 系统监控工具
│   │   ├── user.go         # 用户管理工具
│   │   └── log.go          # 日志分析工具
│   ├── llm/                # LLM 集成
│   │   ├── openai.go       # OpenAI 客户端
│   │   └── prompts.go      # 提示词模板
│   ├── config/             # 配置管理
│   │   └── config.go
│   └── security/           # 安全模块
│       ├── whitelist.go    # 命令白名单
│       └── validator.go    # 输入验证
├── pkg/                    # 可复用包
│   ├── shell/              # Shell 命令执行
│   └── logger/             # 日志系统
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## 核心模块设计

### 1. CLI 交互层 (cmd/)

使用 `cobra` 框架实现命令行交互：

```go
// 交互模式
./arch-agent               # 进入交互式对话
./arch-agent "查询系统状态"  # 单次命令执行

// 管理命令
./arch-agent config        # 配置管理
./arch-agent history       # 历史记录
./arch-agent safe-mode     # 安全模式（只读）
```

**依赖**: `github.com/spf13/cobra`, `github.com/manifoldco/promptui`

### 2. Agent 核心逻辑 (internal/agent/)

#### Agent 主控流程
```go
type Agent struct {
    LLM       *openai.Client
    Tools     *ToolRegistry
    Config    *config.Config
    History   *ConversationHistory
}

func (a *Agent) Process(userInput string) (string, error) {
    // 1. 使用 LLM 理解用户意图
    intent := a.understandIntent(userInput)

    // 2. 规划执行步骤
    plan := a.planExecution(intent)

    // 3. 执行工具调用
    result := a.executeTools(plan)

    // 4. 生成响应
    return a.generateResponse(result)
}
```

#### 工具注册系统
```go
type Tool struct {
    Name        string
    Description string
    Parameters  []Parameter
    Execute     func(ctx context.Context, args map[string]interface{}) (string, error)
    Safe        bool  // 是否安全操作（无需确认）
}

// 工具注册
var ToolRegistry = []Tool{
    // 包管理
    {"pacman_search", "搜索软件包", searchPackage},
    {"pacman_install", "安装软件包（需确认）", installPackage},
    {"pacman_remove", "删除软件包（需确认）", removePackage},
    {"pacman_upgrade", "系统升级（需确认）", upgradeSystem},

    // 服务管理
    {"systemd_list", "列出服务", listServices},
    {"systemd_status", "查看服务状态", serviceStatus},
    {"systemd_start", "启动服务（需确认）", startService},
    {"systemd_stop", "停止服务（需确认）", stopService},

    // 系统监控
    {"monitor_cpu", "CPU 使用率", getCPUUsage},
    {"monitor_memory", "内存使用情况", getMemoryUsage},
    {"monitor_disk", "磁盘使用情况", getDiskUsage},
    {"monitor_process", "进程列表", listProcesses},

    // 日志分析
    {"log_journal", "查询 systemd 日志", queryJournal},

    // 用户管理
    {"user_list", "列出用户", listUsers},
    {"user_info", "用户信息", userInfo},
}
```

### 3. LLM 集成 (internal/llm/)

#### OpenAI GPT 调用
```go
type LLMClient struct {
    client *openai.Client
    model  string  // gpt-4o / gpt-4-turbo
}

func (c *LLMClient) Chat(ctx context.Context, messages []Message, tools []Tool) (Response, error) {
    // 使用 Function Calling
    resp, err := c.client.Chat(ctx, openai.ChatRequest{
        Model:    c.model,
        Messages: messages,
        Tools:    c.convertTools(tools),
    })
    return resp, err
}
```

#### System Prompt 模板
```
你是一个 Arch Linux 系统管理助手，帮助用户完成系统管理和监控任务。

可用工具：
{{.ToolList}}

规则：
1. 只执行用户明确请求的操作
2. 涉及系统变更的操作（安装/删除软件、启停服务）需要告知用户风险
3. 优先使用只读工具获取信息
4. 命令执行失败时分析原因并给出建议
5. 使用简洁专业的中文回复

当前系统信息：
- Arch Linux
- Kernel: {{.KernelVersion}}
- Uptime: {{.Uptime}}
```

### 4. 工具实现 (internal/tools/)

#### 包管理工具 (pacman.go)
```go
func searchPackage(ctx context.Context, name string) ([]Package, error) {
    // 执行 pacman -Ss
    cmd := exec.CommandContext(ctx, "pacman", "-Ss", name)
    output, err := cmd.Output()
    // 解析输出
}

func installPackage(ctx context.Context, name string) error {
    // 执行 sudo pacman -S
    // 注意：需要用户级别权限，可能需要配置无密码 sudo 或使用 yay
}
```

#### Systemd 服务管理 (systemd.go)
```go
func listServices(ctx context.Context) ([]Service, error) {
    // 使用 systemctl list-units
    // 或使用 D-Bus 接口
}

func serviceStatus(ctx context.Context, name string) (ServiceStatus, error) {
    cmd := exec.Command("systemctl", "status", name)
    // 解析输出
}
```

#### 系统监控 (monitor.go)
```go
func getCPUUsage() (float64, error) {
    // 读取 /proc/stat
}

func getMemoryUsage() (MemoryInfo, error) {
    // 读取 /proc/meminfo
}

func getDiskUsage() (map[string]DiskInfo, error) {
    cmd := exec.Command("df", "-h")
}
```

### 5. 安全模块 (internal/security/)

#### 命令白名单
```go
var SafeCommands = map[string]bool{
    "pacman -Ss":    true,
    "pacman -Si":    true,
    "pacman -Q":     true,
    "systemctl status": true,
    "systemctl list-units": true,
    "journalctl":    true,
    "df":           true,
    "free":         true,
    "uptime":       true,
}

var DangerCommands = map[string]bool{
    "pacman -S":    true,  // 需要确认
    "pacman -R":    true,  // 需要确认
    "systemctl start": true,  // 需要确认
    "systemctl stop":  true,  // 需要确认
}
```

#### 输入验证
```go
func ValidateCommand(cmd string) error {
    // 检查命令注入
    if strings.Contains(cmd, "&&") || strings.Contains(cmd, "||") {
        return errors.New("不允许使用命令组合")
    }
    if strings.Contains(cmd, ">") || strings.Contains(cmd, "<") {
        return errors.New("不允许使用重定向")
    }
    // 检查危险字符
    return nil
}
```

---

## 配置管理

### 配置文件结构 (~/.config/arch-agent/config.yaml)
```yaml
# OpenAI 配置
llm:
  api_key: "sk-xxx"
  model: "gpt-4o"
  base_url: "https://api.openai.com/v1"  # 支持自定义 endpoint

# 安全设置
security:
  confirm_before_action: true
  safe_mode: false  # 只读模式
  max_execution_time: 300s

# 日志配置
logging:
  level: "info"
  file: "~/.local/state/arch-agent/agent.log"

# 命令配置
commands:
  pacman_cmd: "pacman"
  yay_cmd: "yay"  # 如果安装了 yay
  sudo_cmd: "sudo"
```

---

## 开发阶段

### Phase 1: 基础框架
- [x] 初始化项目结构
- [ ] 实现 CLI 框架（cobra）
- [ ] 实现配置管理系统
- [ ] 实现日志系统

### Phase 2: LLM 集成
- [ ] OpenAI 客户端封装
- [ ] Function Calling 实现
- [ ] 对话历史管理
- [ ] System Prompt 设计

### Phase 3: 工具实现
- [ ] 包管理工具（pacman/yay）
- [ ] Systemd 服务管理
- [ ] 系统监控工具
- [ ] 日志分析工具

### Phase 4: 安全与优化
- [ ] 命令白名单机制
- [ ] 输入验证
- [ ] 执行超时控制
- [ ] 错误处理和恢复

### Phase 5: 测试与文档
- [ ] 单元测试
- [ ] 集成测试
- [ ] 使用文档
- [ ] 部署指南

---

## 构建和安装

### Makefile
```makefile
.PHONY: build install clean test

BUILD_DIR=bin
BINARY=arch-agent

build:
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd

install: build
	install -Dm755 $(BUILD_DIR)/$(BINARY) ~/.local/bin/$(BINARY)
	install -Dm644 config.yaml ~/.config/arch-agent/config.example.yaml

test:
	go test -v ./...

clean:
	rm -rf $(BUILD_DIR)
```

### AUR 包（可选）
创建 `PKGBUILD` 以便发布到 AUR：
```bash
makepkg -si
```

---

## 依赖列表

```
require (
    github.com/spf13/cobra v1.8.0
    github.com/manifoldco/promptui v0.9.0
    github.com/sashabaranov/go-openai v1.20.0
    gopkg.in/yaml.v3 v3.0.1
)
```

---

## 使用示例

```bash
# 交互模式
$ arch-agent
> 我的系统可以升级吗
Agent: 检测到 12 个可升级的包，包括 linux 6.7.8 -> 6.7.9
> 帮我升级系统
Agent: 即将执行系统升级，这会重启部分服务。确认？[y/N] y
[正在执行 sudo pacman -Syu...]

# 单次命令
$ arch-agent "检查 CPU 温度"
CPU 温度: 45°C (core 0-4 平均)
```

---

## 安全注意事项

1. **不使用 root 权限**：所有命令以当前用户权限执行
2. **sudo 配置**：如需 sudo，配置用户组特定的无密码权限
3. **命令白名单**：严格限制可执行的命令
4. **输入验证**：防止命令注入攻击
5. **审计日志**：记录所有执行的命令和结果
