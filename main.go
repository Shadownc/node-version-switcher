package main

import (
	"context"
	"embed"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/getlantern/systray"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/trayicon.ico
var trayIcon []byte

// AppState 结构体用于管理全局状态
type AppState struct {
	app      *App
	ctx      context.Context
	mu       sync.RWMutex
	quitChan chan struct{}
}

var state = &AppState{
	quitChan: make(chan struct{}),
}

// 安全地设置context
func (s *AppState) SetContext(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ctx = ctx
}

// 安全地获取context
func (s *AppState) GetContext() context.Context {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ctx
}

func main() {
	// 初始化 App
	state.app = NewApp()

	// 创建应用菜单
	appMenu := menu.NewMenu()
	FileMenu := appMenu.AddSubmenu("File")
	FileMenu.AddText("Quit", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
		if ctx := state.GetContext(); ctx != nil {
			runtime.Quit(ctx)
		}
	})

	// 使用channel来确保systray准备就绪
	systrayReady := make(chan struct{})

	// 启动systray
	go func() {
		systray.Run(func() {
			onReady(systrayReady)
		}, onExit)
	}()

	// 等待systray准备就绪
	<-systrayReady

	// 设置debug模式
	if os.Getenv("DEBUG") == "true" {
		state.app.debugMode = true
		fmt.Println("Debug: Debug mode enabled via environment variable")
	}

	// 检查日志文件是否存在
	existingLogFile := false
	if _, err := os.Stat(state.app.logFilePath); err == nil {
		existingLogFile = true
		fmt.Println("Debug: Log file already exists, skipping dialog")
	}

	err := wails.Run(&options.App{
		Title:            "Node Version Switcher",
		Width:            1024,
		Height:           768,
		MinWidth:         1024,
		MinHeight:        768,
		MaxWidth:         4096,
		MaxHeight:        2160,
		DisableResize:    false,
		Fullscreen:       false,
		WindowStartState: options.Normal,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		Menu:             appMenu,
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup: func(ctx context.Context) {
			state.SetContext(ctx)
			fmt.Println("Debug: Application starting up")

			if !existingLogFile && !state.app.debugMode {
				fmt.Println("Debug: Not in debug mode and no existing log file, showing dialog")

				dialogComplete := make(chan bool)

				go func() {
					result, err := runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
						Type:          runtime.QuestionDialog,
						Title:         "日志设置",
						Message:       "是否启用应用程序日志记录？",
						Buttons:       []string{"是", "否"},
						DefaultButton: "否",
						CancelButton:  "否",
					})

					if err != nil {
						fmt.Printf("Debug: Error showing dialog: %v\n", err)
						dialogComplete <- true
						return
					}

					fmt.Printf("Debug: Dialog result: %s\n", result)

					if result == "Yes" {
						fmt.Println("Debug: User selected '是', enabling logs")
						state.app.enableLogs = true
						fmt.Println("Debug: Logging has been enabled")
					} else {
						fmt.Println("Debug: User selected '否', logs will be disabled")
					}

					dialogComplete <- true
				}()

				<-dialogComplete
			} else if existingLogFile || state.app.debugMode {
				state.app.enableLogs = true
				fmt.Println("Debug: Logging automatically enabled due to existing log file or debug mode")
			}

			fmt.Printf("Debug: Final logging state - enableLogs: %v, debugMode: %v\n",
				state.app.enableLogs, state.app.debugMode)

			if state.app.enableLogs {
				fmt.Println("Debug: Attempting to create initial log entry")
				state.app.logToFile("Application startup initiated")
			}

			state.app.startup(ctx)
		},
		OnShutdown:    state.app.shutdown,
		OnBeforeClose: state.app.beforeClose,
		Bind: []interface{}{
			state.app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
			Theme:                windows.SystemDefault,
			CustomTheme: &windows.ThemeSettings{
				DarkModeTitleBar: 1,
			},
		},
	})

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		if state.app.enableLogs {
			state.app.logToFile("启动错误: " + err.Error())
		}
	}
}

func onReady(ready chan<- struct{}) {
	defer close(ready) // 通知主程序systray已准备就绪

	// 设置托盘图标和标题
	systray.SetIcon(trayIcon)
	systray.SetTitle("Node Version Switcher")
	systray.SetTooltip("nvm可视化")

	// 添加托盘菜单项
	showAppMenuItem := systray.AddMenuItem("显示窗口", "Show the application window")
	systray.AddSeparator() // 添加分隔线
	quitMenuItem := systray.AddMenuItem("退出", "Quit the application")

	// 处理菜单点击事件
	go func() {
		for {
			select {
			case <-showAppMenuItem.ClickedCh:
				go func() {
					if ctx := state.GetContext(); ctx != nil {
						runtime.WindowShow(ctx)
						runtime.WindowUnminimise(ctx)
						runtime.WindowSetAlwaysOnTop(ctx, true)
						time.Sleep(time.Millisecond * 100) // 短暂延迟
						runtime.WindowSetAlwaysOnTop(ctx, false)
					}
				}()
			case <-quitMenuItem.ClickedCh:
				go func() {
					// 先隐藏窗口减少视觉延迟
					if ctx := state.GetContext(); ctx != nil {
						runtime.WindowHide(ctx)
					}

					// 退出托盘并等待完成
					quitSystray()

					// 通知主程序可以退出
					close(state.quitChan)

					if ctx := state.GetContext(); ctx != nil {
						runtime.Quit(ctx)
					}
				}()
			}
		}
	}()

	// 窗口状态检测
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			<-ticker.C
			if ctx := state.GetContext(); ctx != nil {
				if runtime.WindowIsMinimised(ctx) {
					runtime.Hide(ctx)
				}
			}
		}
	}()
}

func onExit() {
	// 在这里进行清理工作
	ctx := state.GetContext()
	if ctx != nil && state.app != nil && state.app.enableLogs {
		state.app.logToFile("Application shutting down")
	}
}

func quitSystray() {
	done := make(chan struct{})

	go func() {
		// 清除托盘信息
		systray.SetIcon(nil)
		systray.SetTooltip("")
		systray.SetTitle("")

		// 禁用所有菜单项以防止用户在退出过程中点击
		// 注意：这里我们不再调用 ResetMenu

		systray.Quit()
		close(done)
	}()

	// 等待清理完成或超时
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		// 如果超时，继续执行
	}
}

// minimize 处理最小化事件
func (a *App) minimize(ctx context.Context) {
	if ctx == nil {
		return
	}
	runtime.Hide(ctx)
}
