package agent

import (
	"context"
	"fmt"

	"github.com/npc1607/arch-linux-agent/internal/config"
	"github.com/npc1607/arch-linux-agent/internal/tools"
	"github.com/npc1607/arch-linux-agent/pkg/logger"
)

// LazyExample 演示懒加载功能的使用
func LazyExample(cfg *config.Config, exec *tools.Executor) {
	ctx := context.Background()

	// 1. 创建基础工具注册表
	baseRegistry := &ToolRegistry{
		tools: make(map[string]*Tool),
	}

	// 2. 创建懒加载注册表
	lazyRegistry := NewLazyToolRegistry(baseRegistry)

	// 3. 设置真实的 MCP 工具组
	SetupRealMCPServers(lazyRegistry, cfg, exec)

	// 4. 获取初始工具摘要（不加载所有工具，只返回摘要）
	fmt.Println("\n=== 获取初始工具摘要 ===")
	toolsSummary := lazyRegistry.GetToolSummary()
	for _, tool := range toolsSummary {
		fmt.Printf("- %s: %s\n", tool.Function.Name, tool.Function.Description)
	}

	// 5. 检查消息并自动加载相关工具
	fmt.Println("\n=== 检查用户消息并加载相关工具 ===")
	userMessage := "我在 GitHub 上有个仓库，想搜索一些代码"
	loadedGroups, err := lazyRegistry.CheckAndLoad(ctx, userMessage)

	if IsToolLoadPending(err) {
		groupName, _ := GetGroupNameFromError(err)
		fmt.Printf("⚠️  需要用户确认加载工具组: %s\n", groupName)

		// 模拟用户确认
		fmt.Printf("✅ 用户已确认，加载工具组...\n")
		tools, _ := lazyRegistry.LoadGroup(ctx, groupName, true)
		fmt.Printf("已加载 %d 个工具\n", len(tools))
	} else if err != nil {
		logger.Error("加载失败", logger.Err(err))
	} else {
		fmt.Printf("✅ 自动加载了工具组: %v\n", loadedGroups)
	}

	// 6. 获取已加载的工具
	fmt.Println("\n=== 获取所有已加载的工具 ===")
	loadedTools := lazyRegistry.GetLoadedTools()
	fmt.Printf("当前共有 %d 个工具可用\n", len(loadedTools))

	// 7. 查看工具组状态
	fmt.Println("\n=== 工具组状态 ===")
	status := lazyRegistry.GetGroupStatus()
	formattedStatus := FormatLazyToolsStatus(status)
	fmt.Println(formattedStatus)

	// 8. 模拟使用工具
	fmt.Println("\n=== 模拟执行 GitHub 工具 ===")
	if lazyRegistry.IsGroupLoaded("github") {
		// 执行 GitHub 搜索
		result, err := baseRegistry.Execute(ctx, "github_search_repositories", map[string]interface{}{
			"query": "go-rest-api",
		})
		if err != nil {
			logger.Error("工具执行失败", logger.Err(err))
		} else {
			fmt.Println(result)
		}
	}

	// 9. 测试另一个关键词
	fmt.Println("\n=== 测试 Notion 关键词触发 ===")
	userMessage2 := "帮我查一下 Notion 里的项目文档"
	_, err = lazyRegistry.CheckAndLoad(ctx, userMessage2)

	if IsToolLoadPending(err) {
		groupName, _ := GetGroupNameFromError(err)
		fmt.Printf("⚠️  需要加载: %s\n", groupName)

		// 自动确认并加载
		tools, _ := lazyRegistry.LoadGroup(ctx, groupName, true)
		fmt.Printf("✅ 已加载 %d 个 Notion 工具\n", len(tools))
	}

	// 10. 最终状态
	fmt.Println("\n=== 最终工具组状态 ===")
	status = lazyRegistry.GetGroupStatus()
	fmt.Println(FormatLazyToolsStatus(status))
}

// LazyUsageExample 懒加载使用示例
func LazyUsageExample(cfg *config.Config, exec *tools.Executor) {
	ctx := context.Background()

	// 创建注册表
	lazyRegistry := NewLazyToolRegistry(nil)
	SetupRealMCPServers(lazyRegistry, cfg, exec)

	fmt.Println(`
╔════════════════════════════════════════════════════════════╗
║           MCP 工具懒加载系统 - 使用示例                   ║
╚════════════════════════════════════════════════════════════╝

核心概念:
---------
1. 工具按需加载，不会在启动时全部加载到内存
2. 关键词触发：检测到相关关键词时自动提示加载
3. 用户确认：需要用户确认后才加载工具组
4. 一次加载，永久可用：工具组加载后在会话中保持可用

典型使用场景:
------------

场景 1: 关键词触发加载
-----------------------------
用户: "我在 GitHub 上有个项目想查一下"
      ↓
系统检测到关键词 "github"
      ↓
系统: "检测到需要使用 GitHub 工具，是否加载？ [Y/n]"
      ↓
用户: Y
      ↓
加载 GitHub 工具组（3个工具）

场景 2: 手动加载工具组
-----------------------------
系统: enable_github_tools(confirm=true)
      ↓
加载 GitHub 工具组

场景 3: 查看工具状态
-----------------------------
系统: 显示所有已加载和未加载的工具组
`)

	// 演示关键功能
	fmt.Println("📊 工具组初始状态:")
	status := lazyRegistry.GetGroupStatus()
	fmt.Println(FormatLazyToolsStatus(status))

	// 模拟关键词触发
	fmt.Println("\n🔍 模拟用户输入: '我在 GitHub 上有个项目'")
	_, err := lazyRegistry.CheckAndLoad(ctx, "我在 GitHub 上有个项目")

	if IsToolLoadPending(err) {
		groupName, _ := GetGroupNameFromError(err)
		fmt.Printf("\n💡 系统检测到需要 '%s' 工具组\n", groupName)
		fmt.Println("💡 系统询问: 是否加载？ [Y/n]")
		fmt.Println("✅ 用户确认: Y")

		tools, err := lazyRegistry.LoadGroup(ctx, groupName, true)
		if err != nil {
			logger.Error("加载失败", logger.Err(err))
			return
		}

		fmt.Printf("\n✅ 成功加载 %d 个工具:\n", len(tools))
		for _, tool := range tools {
			fmt.Printf("   - %s: %s\n", tool.Name, tool.Description)
		}
	}

	fmt.Println("\n\n💡 配置示例:")
	fmt.Println(`
{
  "lazy_loading": {
    "enabled": true,
    "auto_load": false,  // false=需要用户确认, true=自动加载
    "max_tools_per_group": 50,
    "groups": {
      "github": {
        "enabled": true,
        "trigger_keywords": ["github", "repo", "仓库", "代码"],
        "description": "GitHub 仓库管理工具",
        "category": "development"
      },
      "notion": {
        "enabled": true,
        "trigger_keywords": ["notion", "文档", "wiki"],
        "description": "Notion 文档工具",
        "category": "documentation"
      }
    }
  }
}
`)

	fmt.Println("\n✨ 优势:")
	fmt.Println(`
- 🚀 快速启动: 不需要等待所有工具加载
- 💾 节省内存: 只加载需要的工具
- 🎯 精准匹配: 关键词智能触发
- 🔒 安全控制: 需要用户确认才加载
- 📊 状态可见: 随时查看工具加载状态
- 🔄 会话持久: 加载后在当前会话保持可用
`)
}
