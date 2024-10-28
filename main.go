package main

import (
	"context"
	"embed"
	"os"
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
var trayIcon []byte // 确保这个文件路径正确，且为小尺寸（16x16 或 32x32）

// 将 app 作为全局变量
var app *App

func main() {
	app = NewApp()

	// 创建应用菜单
	appMenu := menu.NewMenu()
	FileMenu := appMenu.AddSubmenu("File")
	FileMenu.AddText("Quit", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
		runtime.Quit(app.ctx)
	})

	// 使用 systray 库来创建托盘图标
	go func() {
		systray.Run(onReady, onExit)
	}()

	// 如果环境变量设置了 DEBUG=true，则开启调试模式
	if os.Getenv("DEBUG") == "true" {
		app.debugMode = true
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
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		OnBeforeClose:    app.beforeClose,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}

func onReady() {
	// 检查图标是否已嵌入成功
	println("Tray Icon Length:", len(trayIcon)) // 打印嵌入的图标长度，确认是否正确嵌入

	// 设置托盘图标
	systray.SetIcon(trayIcon) // 使用嵌入的 trayIcon 图标
	systray.SetTitle("Node Version Switcher")
	systray.SetTooltip("Manage Node.js versions with ease!")

	// 添加托盘菜单项
	showAppMenuItem := systray.AddMenuItem("显示窗口", "Show the application window")
	quitMenuItem := systray.AddMenuItem("退出", "Quit the application")

	// 启动一个 Goroutine 来监听菜单点击事件
	go func() {
		for {
			select {
			case <-showAppMenuItem.ClickedCh:
				// 使用 Goroutine 处理显示事件，避免阻塞
				go func() {
					runtime.WindowShow(app.ctx) // 显式显示主窗口
				}()
			case <-quitMenuItem.ClickedCh:
				// 使用 Goroutine 处理退出事件
				go func() {
					runtime.Quit(app.ctx)
				}()
			}
		}
	}()

	// 额外的 Goroutine 来检测窗口状态，隐藏到托盘
	go func() {
		for {
			// 使用定时器检查窗口状态，模拟 OnMinimise 的行为
			time.Sleep(1 * time.Second)
			isMinimized := runtime.WindowIsMinimised(app.ctx)
			if isMinimized {
				runtime.Hide(app.ctx) // 如果最小化了，则隐藏窗口到托盘
			}
		}
	}()
}

func onExit() {
	// 清理资源
}

func (a *App) minimize(ctx context.Context) {
	runtime.Hide(ctx) // 最小化时隐藏到系统托盘
}
