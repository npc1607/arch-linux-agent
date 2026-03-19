# Arch Linux AI Agent - CLI 使用指南

## 安装

### 从源码编译

```bash
# 克隆仓库
git clone https://github.com/npc1607/arch-linux-agent.git
cd arch-linux-agent

# 编译
make build

# 安装到本地
make install
```

### 二进制文件

编译后的二进制文件位于 `bin/arch-agent`，可以直接复制到 PATH 中的目录。

## 配置

### API Key 设置

有三种方式设置 OpenAI API Key：

#### 1. 环境变量（推荐）

```bash
export OPENAI_API_KEY="sk-your-key-here"
```

可以添加到 `~/.bashrc` 或 `~/.zshrc`：

```bash
echo 'export OPENAI_API_KEY="sk-your-key-here"' >> ~/.bashrc
source ~/.bashrc
```

#### 2. 配置文件

创建 `~/.config/arch-agent/config.yaml`：

```yaml
api-key: "sk-your-key-here"
model: "gpt-4o"
stream: true
```

参考配置模板：`config.example.yaml`

#### 3. 命令行参数

```bash
arch-agent --api-key "sk-your-key-here" chat
```

### 自定义 API 端点

如果使用兼容 OpenAI API 的服务（如 DeepSeek），可以设置自定义 Base URL：

```bash
arch-agent --base-url "https://api.deepseek.com/v1" chat
```

或在配置文件中设置：

```yaml
base-url: "https://api.deepseek.com/v1"
```

## 使用

### 基本命令

```bash
# 查看帮助
arch-agent --help

# 查看版本
arch-agent --version

# 查看特定命令帮助
arch-agent chat --help
arch-agent ask --help
```

### 交互式对话模式

```bash
# 启动交互模式（默认流式输出）
arch-agent chat

# 禁用流式输出
arch-agent chat --no-stream

# 使用安全模式（只读）
arch-agent chat --safe-mode

# 指定模型
arch-agent chat --model gpt-4-turbo
```

**交互模式命令**：

| 命令 | 说明 |
|------|------|
| `help`, `?` | 显示帮助信息 |
| `clear`, `cls` | 清空对话历史 |
| `exit`, `quit`, `:q` | 退出程序 |
| `Ctrl+C` | 中断当前对话 |

### 单次提问模式

```bash
# 基本用法
arch-agent ask "检查系统状态"

# 禁用流式输出
arch-agent ask --no-stream "为什么 nginx 启动失败？"

# 使用安全模式
arch-agent ask --safe-mode "列出已安装的包"
```

### 默认模式（快捷方式）

```bash
# 直接运行进入交互模式
arch-agent

# 等同于
arch-agent chat
```

## 命令行参数

### 全局参数

| 参数 | 简写 | 默认值 | 说明 |
|------|------|--------|------|
| `--api-key` | - | - | OpenAI API Key |
| `--model` | - | gpt-4o | 使用的模型 |
| `--base-url` | - | - | 自定义 API Base URL |
| `--config` | - | ~/.config/arch-agent/config.yaml | 配置文件路径 |
| `--stream` | - | true | 启用流式输出 |
| `--safe-mode` | - | false | 安全模式（只读） |
| `--max-tokens` | - | 4096 | 最大输出 tokens |
| `--temperature` | - | 0.7 | 温度参数 (0.0-2.0) |
| `--verbose` | -v | false | 详细输出 |

### Chat 命令参数

| 参数 | 说明 |
|------|------|
| `--no-stream` | 禁用流式输出 |

## 示例

### 系统管理

```bash
# 查询系统状态
arch-agent ask "系统可以升级吗？"

# 升级系统
arch-agent ask "帮我执行系统升级"

# 搜索软件包
arch-agent ask "搜索 docker 相关的包"

# 安装软件
arch-agent ask "安装 nginx"
```

### 服务管理

```bash
# 查看服务状态
arch-agent ask "查看 nginx 服务状态"

# 启动服务
arch-agent ask "启动 docker 服务"

# 查看所有服务
arch-agent ask "列出所有运行中的服务"
```

### 系统监控

```bash
# 查看 CPU 使用率
arch-agent ask "CPU 使用率多少？"

# 查看内存使用
arch-agent ask "内存使用情况如何？"

# 查看磁盘空间
arch-agent ask "还有多少磁盘空间？"
```

### 故障排查

```bash
# 查看日志
arch-agent ask "查看最近的系统错误日志"

# 诊断问题
arch-agent ask "为什么网络连接不上？"

# 分析配置
arch-agent ask "检查 nginx 配置是否有问题"
```

## 高级用法

### 安全模式

安全模式下，Agent 只执行只读操作：

```bash
arch-agent chat --safe-mode
```

允许的操作：
- `pacman -Ss` (搜索包)
- `systemctl status` (查看状态)
- `df -h`, `free -h` (查看资源)
- `journalctl` (查看日志)

不允许的操作：
- `pacman -S` (安装包)
- `systemctl start/stop` (启停服务)
- 任何修改系统的操作

### 流式输出

流式输出（默认）：逐字显示响应，类似 ChatGPT

```bash
arch-agent chat  # 默认启用流式输出
```

非流式输出：等待完整响应后一次性显示

```bash
arch-agent chat --no-stream
arch-agent --stream=false chat
```

### 温度参数

控制响应的随机性：

```bash
# 低温度（更确定性）
arch-agent ask --temperature 0.2 "检查系统"

# 高温度（更创造性）
arch-agent ask --temperature 1.5 "给我一些建议"
```

### 最大 Tokens

限制响应长度：

```bash
arch-agent ask --max-tokens 1000 "简要说明"
```

## 配置文件

配置文件位置：`~/.config/arch-agent/config.yaml`

```yaml
# OpenAI 配置
api-key: "sk-your-key-here"
model: "gpt-4o"  # gpt-4o, gpt-4-turbo, gpt-3.5-turbo
base-url: ""  # 可选：自定义 API 端点

# 行为配置
stream: true  # 启用流式输出
safe-mode: false  # 安全模式

# LLM 参数
max-tokens: 4096
temperature: 0.7

# 日志配置
verbose: false
```

## 开发

```bash
# 安装依赖
make deps

# 运行测试
make test

# 代码检查
make lint

# 开发模式运行
make dev
```

## 故障排查

### API Key 错误

```
错误: 未设置 API Key
```

解决：设置环境变量或配置文件

```bash
export OPENAI_API_KEY="sk-your-key-here"
```

### 网络错误

```
创建流式请求失败: ...
```

解决：检查网络连接，或使用代理

```bash
export HTTP_PROXY=http://127.0.0.1:7890
export HTTPS_PROXY=http://127.0.0.1:7890
```

### 模型不可用

```
错误: 模型不可用
```

解决：切换到可用模型

```bash
arch-agent chat --model gpt-3.5-turbo
```
