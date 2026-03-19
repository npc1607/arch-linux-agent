package tools

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/npc1607/arch-linux-agent/pkg/logger"
)

// PacmanTool 包管理工具
type PacmanTool struct {
	exec    *Executor
	cmdName string
}

// NewPacmanTool 创建包管理工具
func NewPacmanTool(exec *Executor, cmdName string) *PacmanTool {
	if cmdName == "" {
		cmdName = "pacman"
	}
	return &PacmanTool{
		exec:    exec,
		cmdName: cmdName,
	}
}

// Package 包信息
type Package struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Repository  string `json:"repository"`
	Size        string `json:"size"`
	Installed   bool   `json:"installed"`
}

// Search 搜索软件包
func (p *PacmanTool) Search(ctx context.Context, keyword string) ([]Package, error) {
	logger.Info("搜索软件包", logger.String("keyword", keyword))

	result, err := p.exec.Run(ctx, p.cmdName, "-Ss", keyword)
	if err != nil {
		return nil, fmt.Errorf("搜索失败: %w", err)
	}

	packages := p.parseSearchResult(result.Output)
	logger.Info("搜索完成", logger.Int("count", len(packages)))

	return packages, nil
}

// parseSearchResult 解析搜索结果
func (p *PacmanTool) parseSearchResult(output string) []Package {
	var packages []Package
	lines := strings.Split(output, "\n")

	var currentPkg *Package
	repoRe := regexp.MustCompile(`^([^/]+)/([^ ]+) (.+)$`)
	infoRe := regexp.MustCompile(`^    (.+)$`)

	for _, line := range lines {
		if matches := repoRe.FindStringSubmatch(line); len(matches) > 0 {
			if currentPkg != nil {
				packages = append(packages, *currentPkg)
			}
			currentPkg = &Package{
				Repository:  matches[1],
				Name:        matches[2],
				Description: matches[3],
			}
		} else if currentPkg != nil {
			if matches := infoRe.FindStringSubmatch(line); len(matches) > 0 {
				currentPkg.Description += " " + matches[1]
			}
		}
	}

	if currentPkg != nil {
		packages = append(packages, *currentPkg)
	}

	return packages
}

// Query 查询已安装的包
func (p *PacmanTool) Query(ctx context.Context, packageName string) (*Package, error) {
	logger.Info("查询软件包", logger.String("package", packageName))

	result, err := p.exec.Run(ctx, p.cmdName, "-Qi", packageName)
	if err != nil {
		return nil, fmt.Errorf("查询失败: %w", err)
	}

	pkg := p.parseQueryResult(result.Output)
	if pkg != nil {
		pkg.Installed = true
	}

	return pkg, nil
}

// parseQueryResult 解析查询结果
func (p *PacmanTool) parseQueryResult(output string) *Package {
	pkg := &Package{}
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Name") {
			pkg.Name = strings.TrimSpace(strings.TrimPrefix(line, "Name"))
		} else if strings.HasPrefix(line, "Version") {
			pkg.Version = strings.TrimSpace(strings.TrimPrefix(line, "Version"))
		} else if strings.HasPrefix(line, "Description") {
			pkg.Description = strings.TrimSpace(strings.TrimPrefix(line, "Description"))
		} else if strings.HasPrefix(line, "Repository") {
			pkg.Repository = strings.TrimSpace(strings.TrimPrefix(line, "Repository"))
		} else if strings.HasPrefix(line, "Installed Size") {
			pkg.Size = strings.TrimSpace(strings.TrimPrefix(line, "Installed Size"))
		}
	}

	return pkg
}

// ListInstalled 列出已安装的包
func (p *PacmanTool) ListInstalled(ctx context.Context) ([]Package, error) {
	logger.Info("列出已安装的包")

	result, err := p.exec.Run(ctx, p.cmdName, "-Q")
	if err != nil {
		return nil, fmt.Errorf("列出失败: %w", err)
	}

	packages := p.parseListResult(result.Output)
	logger.Info("列出完成", logger.Int("count", len(packages)))

	return packages, nil
}

// parseListResult 解析列表结果
func (p *PacmanTool) parseListResult(output string) []Package {
	var packages []Package
	lines := strings.Split(output, "\n")

	pattern := regexp.MustCompile(`^([^ ]+) ([^ ]+)$`)
	for _, line := range lines {
		if matches := pattern.FindStringSubmatch(line); len(matches) > 0 {
			packages = append(packages, Package{
				Name:    matches[1],
				Version: matches[2],
				Installed: true,
			})
		}
	}

	return packages
}

// CheckUpdates 检查可用更新
func (p *PacmanTool) CheckUpdates(ctx context.Context) ([]Package, error) {
	logger.Info("检查系统更新")

	// 尝试同步数据库（需要root权限）
	_, syncErr := p.exec.Run(ctx, p.cmdName, "-Sy")
	if syncErr != nil {
		logger.Warn("同步数据库失败（可能需要root权限），将使用现有数据库检查", logger.Err(syncErr))
	}

	// 检查更新
	result, err := p.exec.Run(ctx, p.cmdName, "-Qu")
	if err != nil {
		// pacman -Qu 在没有更新时返回错误，但这是正常情况
		// 检查是否是因为没有更新导致的
		if result.ExitCode != 0 && result.Output == "" {
			// 没有可用更新，返回空列表
			logger.Info("检查完成", logger.Int("updates", 0))
			return []Package{}, nil
		}
		return nil, fmt.Errorf("检查更新失败: %w", err)
	}

	packages := p.parseListResult(result.Output)
	logger.Info("检查完成", logger.Int("updates", len(packages)))

	return packages, nil
}

// Install 安装软件包（需要确认）
func (p *PacmanTool) Install(ctx context.Context, packages []string, noConfirm bool) (*CommandResult, error) {
	if len(packages) == 0 {
		return nil, fmt.Errorf("没有指定要安装的包")
	}

	logger.Info("准备安装软件包",
		logger.Any("packages", packages),
		logger.Bool("no_confirm", noConfirm),
	)

	args := []string{"-S"}
	if noConfirm {
		args = append(args, "--noconfirm")
	}
	args = append(args, packages...)

	result, err := p.exec.Run(ctx, p.cmdName, args...)
	if err != nil {
		logger.Error("安装失败", logger.Err(err))
	} else {
		logger.Info("安装成功")
	}

	return result, err
}

// Remove 删除软件包
func (p *PacmanTool) Remove(ctx context.Context, packages []string, noConfirm bool) (*CommandResult, error) {
	if len(packages) == 0 {
		return nil, fmt.Errorf("没有指定要删除的包")
	}

	logger.Info("准备删除软件包",
		logger.Any("packages", packages),
	)

	args := []string{"-R"}
	if noConfirm {
		args = append(args, "--noconfirm")
	}
	args = append(args, packages...)

	result, err := p.exec.Run(ctx, p.cmdName, args...)
	if err != nil {
		logger.Error("删除失败", logger.Err(err))
	} else {
		logger.Info("删除成功")
	}

	return result, err
}

// Upgrade 系统升级
func (p *PacmanTool) Upgrade(ctx context.Context, noConfirm bool) (*CommandResult, error) {
	logger.Info("准备升级系统", logger.Bool("no_confirm", noConfirm))

	args := []string{"-Syu"}
	if noConfirm {
		args = append(args, "--noconfirm")
	}

	result, err := p.exec.Run(ctx, p.cmdName, args...)
	if err != nil {
		logger.Error("升级失败", logger.Err(err))
	} else {
		logger.Info("升级成功")
	}

	return result, err
}

// GetUpgradeableSize 估算升级大小
func (p *PacmanTool) GetUpgradeableSize(ctx context.Context) (string, error) {
	result, err := p.exec.Run(ctx, p.cmdName, "-Syu", "--print-format", "%s")
	if err != nil {
		return "", err
	}

	// 解析输出获取大小
	return result.Output, nil
}

// Sync 同步包数据库
func (p *PacmanTool) Sync(ctx context.Context) error {
	logger.Info("同步包数据库")

	_, err := p.exec.Run(ctx, p.cmdName, "-Sy")
	if err != nil {
		logger.Error("同步失败", logger.Err(err))
		return err
	}

	logger.Info("同步成功")
	return nil
}
