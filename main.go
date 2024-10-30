package main

import (
	"context"
	"embed"
	"fmt"
	"os"

	"github.com/getlantern/systray"
	"github.com/skratchdot/open-golang/open"
	"github.com/wailsapp/wails/v2"
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
	app    *App
	ctx    context.Context
	cancel context.CancelFunc
}

var state = NewAppState()

// NewAppState creates a new instance of AppState
// NewAppState 创建一个新的 AppState 实例
func NewAppState() *AppState {
	ctx, cancel := context.WithCancel(context.Background())
	return &AppState{
		ctx:    ctx,
		cancel: cancel,
	}
}

func main() {
	// Initialize the App
	// 初始化 App
	state.app = NewApp()

	// Start the system tray in a separate goroutine
	// 启动托盘图标，运行在一个独立的 goroutine 中
	go func() {
		runSystray()
	}()

	// Set debug mode if environment variable is set
	// 如果环境变量设置了DEBUG，则进入调试模式
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

	// Run the Wails application
	// 运行 Wails 应用程序
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
		// Menu:             appMenu,
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup: func(ctx context.Context) {
			state.ctx = ctx // 存储 Wails 提供的上下文以便托盘操作使用
			fmt.Println("Debug: Application starting up")

			// Display dialog for logging setup if not in debug mode and no existing log file
			// 如果不在调试模式且没有现存日志文件，显示日志设置对话框
			if !existingLogFile && !state.app.debugMode {
				fmt.Println("Debug: Not in debug mode and no existing log file, showing dialog")

				dialogComplete := make(chan bool)

				go func() {
					result, err := runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
						Type:          runtime.QuestionDialog,
						Title:         "日志设置",
						Message:       "是否启用应用程序日志记录？",
						Buttons:       []string{"Yes", "No"},
						DefaultButton: "No",
						CancelButton:  "No",
					})

					if err != nil {
						fmt.Printf("Debug: Error showing dialog: %v\n", err)
						dialogComplete <- true
						return
					}

					fmt.Printf("Debug: Dialog result: %s\n", result)

					if result == "Yes" {
						fmt.Println("Debug: User selected 'Yes', enabling logs")
						state.app.enableLogs = true
						fmt.Println("Debug: Logging has been enabled")
					} else {
						fmt.Println("Debug: User selected 'No', logs will be disabled")
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
			return state.app.beforeClose()
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

// runSystray runs the systray
// runSystray 运行托盘图标
func runSystray() {
	systray.Run(onReady, onExit)
}

// onReady is the callback when the system tray icon is ready
// onReady 是托盘图标就绪后的回调
func onReady() {
	go func() {
		systray.SetTemplateIcon(trayIcon, trayIcon)
		systray.SetTitle("Node Version Switcher")
		systray.SetTooltip("nvm可视化")
		blog := systray.AddMenuItem("博客", "Blog")
		github := systray.AddMenuItem("Github", "Github")
		mShow := systray.AddMenuItem("显示应用", "mShow")
		mQuit := systray.AddMenuItem("退出", "Quit")
		for {
			select {
			case <-blog.ClickedCh:
				open.Run("https://blog.lmyself.top")
			case <-github.ClickedCh:
				open.Run("https://github.com/Shadownc/node-version-switcher")
			case <-mShow.ClickedCh:
				// 显示应用窗口
				// 使用 Wails 提供的 runtime API 来显示应用窗口
				fmt.Println("Debug: User clicked 'Show Application'")
				if state.ctx != nil {
					runtime.WindowShow(state.ctx)
					fmt.Println("Debug: Application window shown successfully")
				} else {
					fmt.Println("Debug: Context is nil, cannot show window")
				}
			case <-mQuit.ClickedCh:
				// 退出应用
				fmt.Println("Debug: User clicked 'Quit', shutting down application")
				systray.Quit()                // 关闭托盘图标
				state.app.shutdown(state.ctx) // 调用 app 的 shutdown 以确保应用程序关闭
				os.Exit(0)                    // 完全退出程序
			}
		}
	}()
}

// onExit is the callback when the tray application exits
// onExit 是托盘程序退出时的回调
func onExit() {
	fmt.Println("Debug: onExit called, application is shutting down")
	if state.app != nil && state.app.enableLogs {
		state.app.logToFile("Application shutting down")
	}
	fmt.Println("Debug: Logs have been saved, exit complete")
}
