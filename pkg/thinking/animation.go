package thinking

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"
)

// AnimationType 动画类型
type AnimationType int

const (
	// Dots 点状动画
	Dots AnimationType = iota
	// Spinner 旋转动画
	Spinner
	// Progress 进度条动画
	Progress
	// Text 文本动画
	Text
)

var (
	dotsFrames    = []string{".  ", ".. ", "..."}
	spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	progressFrames = []string{"[          ]", "[=         ]", "[==        ]", "[===       ]",
		"[====      ]", "[=====     ]", "[======    ]", "[=======   ]",
		"[========  ]", "[========= ]", "[==========]"}
	textFrames = []string{"Thinking...", "Processing...", "Analyzing..."}
)

// Animation 动画实例
type Animation struct {
	type_     AnimationType
	frames    []string
	message   string
	stopChan  chan struct{}
	done      chan struct{}
	mu        sync.Mutex
	running   bool
	delay     time.Duration
	started   bool
	ctx       context.Context
	cancelFn  context.CancelFunc
}

// Config 动画配置
type Config struct {
	Type    AnimationType
	Message string
	Delay   time.Duration
}

// New 创建新的动画实例
func New(cfg Config) *Animation {
	frames := getFrames(cfg.Type)
	if cfg.Delay == 0 {
		cfg.Delay = 200 * time.Millisecond
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Animation{
		type_:    cfg.Type,
		frames:   frames,
		message:  cfg.Message,
		stopChan: make(chan struct{}),
		done:     make(chan struct{}),
		delay:    cfg.Delay,
		ctx:      ctx,
		cancelFn: cancel,
	}
}

// getFrames 根据类型获取动画帧
func getFrames(t AnimationType) []string {
	switch t {
	case Dots:
		return dotsFrames
	case Spinner:
		return spinnerFrames
	case Progress:
		return progressFrames
	case Text:
		return textFrames
	default:
		return dotsFrames
	}
}

// Start 启动动画（带延迟显示）
func (a *Animation) Start(delay time.Duration) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.running {
		return
	}

	a.running = true

	// 如果设置了延迟，先等待
	if delay > 0 {
		timer := time.NewTimer(delay)
		select {
		case <-timer.C:
			// 延迟结束，开始显示动画
		case <-a.stopChan:
			timer.Stop()
			a.running = false
			return
		case <-a.ctx.Done():
			timer.Stop()
			a.running = false
			return
		}
	}

	// 启动动画循环
	go a.animate()
	a.started = true
}

// StartImmediate 立即启动动画
func (a *Animation) StartImmediate() {
	a.Start(0)
}

// Stop 停止动画
func (a *Animation) Stop() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.running {
		return
	}

	a.cancelFn()
	close(a.stopChan)

	// 等待动画循环结束
	if a.started {
		select {
		case <-a.done:
		case <-time.After(100 * time.Millisecond):
			// 超时保护
		}
	}

	// 清除当前行
	if a.started {
		a.clearLine()
	}

	a.running = false
	a.started = false

	// 重置状态
	a.stopChan = make(chan struct{})
	a.done = make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	a.ctx = ctx
	a.cancelFn = cancel
}

// animate 动画循环
func (a *Animation) animate() {
	defer close(a.done)

	ticker := time.NewTicker(a.delay)
	defer ticker.Stop()

	frameIndex := 0

	for {
		select {
		case <-ticker.C:
			// 显示当前帧
			frame := a.frames[frameIndex]
			a.displayFrame(frame)
			frameIndex = (frameIndex + 1) % len(a.frames)

		case <-a.stopChan:
			return

		case <-a.ctx.Done():
			return
		}
	}
}

// displayFrame 显示动画帧
func (a *Animation) displayFrame(frame string) {
	// 使用 \r 回到行首，覆盖之前的内容
	if a.message != "" {
		fmt.Fprintf(os.Stderr, "\r%s %s", a.message, frame)
	} else {
		fmt.Fprintf(os.Stderr, "\r%s", frame)
	}
}

// clearLine 清除当前行
func (a *Animation) clearLine() {
	// 清除整行：\r 回到行首，然后输出足够多的空格，再回到行首
	fmt.Fprintf(os.Stderr, "\r%80s\r", " ")
}

// IsRunning 检查动画是否正在运行
func (a *Animation) IsRunning() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.running
}

// SetMessage 设置动画消息
func (a *Animation) SetMessage(msg string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.message = msg
}

// SetType 设置动画类型
func (a *Animation) SetType(t AnimationType) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.type_ = t
	a.frames = getFrames(t)
}

// StartWithTimeout 启动动画，但只在指定时间内显示
// 如果在超时前停止，则不会显示动画
func (a *Animation) StartWithTimeout(timeout time.Duration) {
	go func() {
		select {
		case <-time.After(timeout):
			if a.IsRunning() && !a.started {
				a.mu.Lock()
				if !a.started {
					a.started = true
					go a.animate()
				}
				a.mu.Unlock()
			}
		case <-a.stopChan:
			return
		case <-a.ctx.Done():
			return
		}
	}()

	a.mu.Lock()
	a.running = true
	a.mu.Unlock()
}
