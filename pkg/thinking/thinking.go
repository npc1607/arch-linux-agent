package thinking

import (
	"context"
	"time"
)

// Manager 管理动画生命周期
type Manager struct {
	animation *Animation
	delay     time.Duration
}

// NewManager 创建动画管理器
func NewManager(typ AnimationType, message string) *Manager {
	return &Manager{
		animation: New(Config{
			Type:    typ,
			Message: message,
		}),
		delay: 500 * time.Millisecond, // 默认500ms后才显示动画，避免快速响应时闪烁
	}
}

// NewManagerWithDelay 创建带自定义延迟的动画管理器
func NewManagerWithDelay(typ AnimationType, message string, delay time.Duration) *Manager {
	return &Manager{
		animation: New(Config{
			Type:    typ,
			Message: message,
		}),
		delay: delay,
	}
}

// Start 启动动画（带延迟）
func (m *Manager) Start() {
	m.animation.Start(m.delay)
}

// StartImmediate 立即启动动画
func (m *Manager) StartImmediate() {
	m.animation.StartImmediate()
}

// Stop 停止动画
func (m *Manager) Stop() {
	m.animation.Stop()
}

// Run 运行动画并执行函数
// 如果fn在延迟时间内完成，则不会显示动画
func (m *Manager) Run(ctx context.Context, fn func() error) error {
	m.Start()
	defer m.Stop()

	return fn()
}

// IsRunning 检查动画是否正在运行
func (m *Manager) IsRunning() bool {
	return m.animation.IsRunning()
}

// SetMessage 设置动画消息
func (m *Manager) SetMessage(msg string) {
	m.animation.SetMessage(msg)
}

// SetType 设置动画类型
func (m *Manager) SetType(typ AnimationType) {
	m.animation.SetType(typ)
}

// 便捷函数

// NewSpinner 创建旋转动画
func NewSpinner(message string) *Manager {
	return NewManager(Spinner, message)
}

// NewDots 创建点状动画
func NewDots(message string) *Manager {
	return NewManager(Dots, message)
}

// NewProgress 创建进度条动画
func NewProgress(message string) *Manager {
	return NewManager(Progress, message)
}

// NewText 创建文本动画
func NewText(message string) *Manager {
	return NewManager(Text, message)
}

// NewDefault 创建默认动画（Spinner）
func NewDefault() *Manager {
	return NewManager(Spinner, "")
}
