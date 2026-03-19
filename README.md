# Arch Linux AI Agent

一个基于 Go 语言的 AI Agent，通过自然语言交互来管理和监控 Arch Linux 系统。

## 功能

- 🔧 **系统管理**：pacman/yay 包管理、systemd 服务管理
- 📊 **系统监控**：CPU/内存/磁盘监控、进程监控
- 📝 **日志分析**：systemd 日志查询和分析
- 🤖 **智能助手**：基于 OpenAI GPT 的自然语言交互

## 安装

```bash
go install github.com/npc1607/arch-linux-agent@latest
```

## 使用

```bash
# 交互模式
arch-agent

# 单次命令
arch-agent "检查系统状态"
```

## 配置

配置文件位于 `~/.config/arch-agent/config.yaml`

## 开发

详见 [PLAN.md](./PLAN.md)

## License

MIT
