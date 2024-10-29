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

// AppState struct is used to manage global state
// AppState 结构体用于管理全局状态
type AppState struct {
	app        *App
	ctx        context.Context
	cancel     context.CancelFunc
	mu         sync.RWMutex
	quitChan   chan struct{}
	isQuitting bool
}

// NewAppState creates a new instance of AppState
// NewAppState 创建一个新的 AppState 实例
func NewAppState() *AppState {
	ctx, cancel := context.WithCancel(context.Background())
	return &AppState{
		ctx:      ctx,
		cancel:   cancel,
		quitChan: make(chan struct{}),
	}
}

var state = NewAppState()

// SetContext sets the application context
// SetContext 设置应用程序上下文
func (s *AppState) SetContext(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ctx = ctx
}

// GetContext retrieves the application context
// GetContext 获取应用程序上下文
func (s *AppState) GetContext() context.Context {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ctx
}

// markQuitting marks the application is quitting
// markQuitting 标记应用程序正在退出
func (s *AppState) markQuitting() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isQuitting = true
}

// isInQuitting checks if the application is in quitting state
// isInQuitting 判断应用程序是否在退出中
func (s *AppState) isInQuitting() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isQuitting
}

func main() {
	// Initialize the App
	// 初始化 App
	state.app = NewApp()

	appMenu := menu.NewMenu()
	FileMenu := appMenu.AddSubmenu("File")
	FileMenu.AddText("Quit", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
		if ctx := state.GetContext(); ctx != nil {
			runtime.Quit(ctx)
		}
	})

	go func() {
		systray.Run(onReady, onExit)
	}()

	// Set debug mode
	// 设置debug模式
	if os.Getenv("DEBUG") == "true" {
		state.app.debugMode = true
		fmt.Println("Debug: Debug mode enabled via environment variable")
	}

	// Check if the log file exists
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

					if result == "Yes" { // 保持原功能不变
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
		OnShutdown: state.app.shutdown,
		OnBeforeClose: func(ctx context.Context) bool {
			if state.isInQuitting() {
				return false
			}
			return state.app.beforeClose(ctx)
		},
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

// onReady is the callback when the system tray icon is ready
// onReady 是托盘图标就绪后的回调
func onReady() {
	systray.SetIcon(trayIcon)
	systray.SetTitle("Node Version Switcher")
	systray.SetTooltip("nvm可视化")

	// Right-click menu items
	// 右键菜单项
	showAppMenuItem := systray.AddMenuItem("显示窗口", "Show the application window")
	systray.AddSeparator()
	quitMenuItem := systray.AddMenuItem("退出", "Quit the application")

	// Hidden menu item used to capture left-click events
	// 隐藏的菜单项，用于捕获左键单击事件
	mLeftClick := systray.AddMenuItem("", "")
	mLeftClick.Hide()

	// Left-click to directly show the window
	// 左键单击直接显示窗口
	go func() {
		for {
			select {
			case <-mLeftClick.ClickedCh: // 左键点击直接显示窗口
				showWindow()
			case <-showAppMenuItem.ClickedCh: // 右键菜单的“显示窗口”
				showWindow()
			case <-quitMenuItem.ClickedCh: // 右键菜单的“退出”
				gracefulQuit()
				return
			}
		}
	}()
}

// showWindow function to add retry mechanism
// showWindow 函数，添加重试机制
func showWindow() {
	if appCtx := state.GetContext(); appCtx != nil && !state.isInQuitting() {
		go func() {
			for i := 0; i < 3; i++ { // 最多尝试三次 // Retry up to three times
				runtime.WindowShow(appCtx)
				runtime.WindowUnminimise(appCtx)
				runtime.WindowSetAlwaysOnTop(appCtx, true)
				time.Sleep(time.Millisecond * 100)
				runtime.WindowSetAlwaysOnTop(appCtx, false)
				// 移除错误的判断，保持原有逻辑功能不变
				break
			}
		}()
	}
}

// gracefulQuit gracefully quits the tray application
// gracefulQuit 实现托盘应用程序的优雅退出
func gracefulQuit() {
	state.markQuitting()
	if appCtx := state.GetContext(); appCtx != nil {
		runtime.WindowHide(appCtx)
	}
	systray.Quit()
	if appCtx := state.GetContext(); appCtx != nil {
		runtime.Quit(appCtx)
	}
}

// onExit is the callback when the tray application exits
// onExit 是托盘程序退出时的回调
func onExit() {
	ctx := state.GetContext()
	if ctx != nil && state.app != nil && state.app.enableLogs {
		state.app.logToFile("Application shutting down")
	}
}
