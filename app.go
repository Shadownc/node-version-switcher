package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// NodeVersionInfo represents detailed information for each Node version
// NodeVersionInfo 表示每个 Node.js 版本的详细信息
type NodeVersionInfo struct {
	Version    string
	Status     string
	NpmVersion string // 新增字段，表示 npm 版本
}

// NodeVersion represents an installed Node.js version
// NodeVersion 表示已安装的 Node.js 版本
type NodeVersion struct {
	Version   string
	IsCurrent bool
}

// NodeAPIResponse represents the structure from Node.js API response
// NodeAPIResponse 表示从 Node.js API 获取的版本信息结构
type NodeAPIResponse struct {
	Version string      `json:"version"`
	Date    string      `json:"date"`
	Files   []string    `json:"files"`
	LTS     interface{} `json:"lts"` // 修改为 interface{} 来处理不同的数据类型
	Npm     string      `json:"npm"`
}

// App struct represents the main application
// App 结构体表示主应用程序
type App struct {
	ctx         context.Context
	cancel      context.CancelFunc
	debugMode   bool
	enableLogs  bool
	logFilePath string
	lastActive  time.Time
	mu          sync.RWMutex
}

// NewApp creates a new App application struct
// NewApp 创建一个新的 App 应用程序结构体
func NewApp() *App {
	// 获取可执行文件所在目录
	// Get the directory of the executable file
	execPath, err := os.Executable()
	logPath := "nvm-switcher.log"
	if err == nil {
		logPath = filepath.Join(filepath.Dir(execPath), "nvm-switcher.log")
	}

	return &App{
		debugMode:   false,
		enableLogs:  false,
		logFilePath: logPath,
		lastActive:  time.Now(),
	}
}

// updateLastActive updates the last active timestamp for the application
// updateLastActive 更新应用程序的最后活动时间戳
func (a *App) updateLastActive() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.lastActive = time.Now()
}

// checkHealth checks if the application has been active recently
// checkHealth 检查应用程序是否在最近处于活动状态
func (a *App) checkHealth() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return time.Since(a.lastActive) < time.Minute*5
}

// startup initializes the application when it starts
// startup 在应用程序启动时初始化
func (a *App) startup(ctx context.Context) {
	a.mu.Lock()
	a.ctx, a.cancel = context.WithCancel(ctx)
	a.mu.Unlock()

	a.updateLastActive()
	a.logToFile("Application started")

	// 启动健康检查
	// Start health check
	go a.healthCheck()
}

// healthCheck periodically checks if the application is still healthy
// healthCheck 定期检查应用程序是否仍然健康
func (a *App) healthCheck() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			if !a.checkHealth() {
				a.logToFile("Health check failed - attempting recovery")
				runtime.WindowReload(a.ctx) // 重新加载窗口
			}
			a.updateLastActive()
		}
	}
}

// shutdown cleans up resources when the application shuts down
// shutdown 在应用程序关闭时清理资源
func (a *App) shutdown(ctx context.Context) {
	a.mu.Lock()
	if a.cancel != nil {
		a.cancel()
	}
	a.mu.Unlock()

	a.logToFile("Application shutting down")
}

// beforeClose is called before the application is closed
// beforeClose 在应用程序关闭前调用
func (a *App) beforeClose() bool {
	if a.enableLogs {
		a.logToFile("Application closing initiated")
	}
	return false
}

// logToFile logs a message to the specified log file if logging is enabled
// logToFile 如果启用了日志记录，则将消息记录到指定的日志文件中
func (a *App) logToFile(message string) {
	if !a.enableLogs {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	logDir := filepath.Dir(a.logFilePath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Printf("Failed to create log directory: %v\n", err)
		userHome, err := os.UserHomeDir()
		if err == nil {
			a.logFilePath = filepath.Join(userHome, "nvm-switcher.log")
			logDir = userHome
		} else {
			return
		}
	}

	f, err := os.OpenFile(a.logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	timeStamp := time.Now().Format("2006-01-02 15:04:05")
	logMessage := fmt.Sprintf("[%s] %s\n", timeStamp, message)
	f.Write([]byte(logMessage))
}

// executeNvmCommand runs the specified NVM command with provided arguments and returns the output
// executeNvmCommand 运行指定的 NVM 命令并返回其输出
func (a *App) executeNvmCommand(args ...string) ([]byte, error) {
	a.updateLastActive()

	// 通过 cmd 来运行 nvm
	cmd := exec.Command("cmd", append([]string{"/c", "nvm"}, args...)...)

	// 继承当前环境变量
	cmd.Env = os.Environ()

	// 如果不处于调试模式则隐藏窗口
	if !a.debugMode {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow: true,
		}
	}

	// 获取输出结果
	output, err := cmd.CombinedOutput()
	if err != nil {
		a.logToFile(fmt.Sprintf("Command failed: nvm %v\nError: %v\nOutput: %s\n",
			args, err, string(output)))
	}

	return output, err
}

// InstallNodeVersion installs the specified Node.js version
// InstallNodeVersion 安装指定的 Node.js 版本
func (a *App) InstallNodeVersion(version string) string {
	a.logToFile(fmt.Sprintf("Attempting to install Node.js version: %s", version))
	output, err := a.executeNvmCommand("install", version)
	if err != nil {
		errMsg := fmt.Sprintf("Error installing Node.js %s: %s", version, string(output))
		a.logToFile(errMsg)
		return errMsg
	}
	successMsg := fmt.Sprintf("Successfully installed Node.js %s", version)
	a.logToFile(successMsg)
	return successMsg
}

// UninstallNodeVersion uninstalls the specified Node.js version
// UninstallNodeVersion 卸载指定的 Node.js 版本
func (a *App) UninstallNodeVersion(version string) string {
	a.logToFile(fmt.Sprintf("Attempting to uninstall Node.js version: %s", version))
	output, err := a.executeNvmCommand("uninstall", version)
	if err != nil {
		errMsg := fmt.Sprintf("Error uninstalling Node.js %s: %s", version, string(output))
		a.logToFile(errMsg)
		return errMsg
	}
	successMsg := fmt.Sprintf("Successfully uninstalled Node.js %s", version)
	a.logToFile(successMsg)
	return successMsg
}

// SwitchNodeVersion switches to the specified Node.js version
// SwitchNodeVersion 切换到指定的 Node.js 版本
func (a *App) SwitchNodeVersion(version string) string {
	a.logToFile(fmt.Sprintf("Attempting to switch to Node.js version: %s", version))
	output, err := a.executeNvmCommand("use", version)
	if err != nil {
		errMsg := fmt.Sprintf("Error switching to Node.js %s: %s", version, string(output))
		a.logToFile(errMsg)
		return errMsg
	}

	successMsg := fmt.Sprintf("Successfully switched to Node.js %s", version)
	a.logToFile(successMsg)
	return successMsg
}

// GetInstalledNodeVersions retrieves the Node.js versions installed via NVM on the system
// GetInstalledNodeVersions 获取系统上通过 NVM 安装的 Node.js 版本
func (a *App) GetInstalledNodeVersions() ([]NodeVersion, error) {
	a.logToFile("Fetching installed Node.js versions")
	output, err := a.executeNvmCommand("ls")
	if err != nil {
		a.logToFile(fmt.Sprintf("Error fetching installed versions: %v", err))
		return nil, fmt.Errorf("Error fetching installed versions: %v", err)
	}

	lines := strings.Split(string(output), "\n")
	var versions []NodeVersion
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "->") || strings.Contains(line, "default") || strings.Contains(line, "system") {
			continue
		}

		isCurrent := false
		if strings.HasPrefix(line, "*") {
			isCurrent = true
			line = strings.Replace(line, "*", "", 1)
			line = strings.TrimSpace(line)
		}

		version := strings.Fields(line)[0]
		versions = append(versions, NodeVersion{Version: version, IsCurrent: isCurrent})
	}

	if a.debugMode {
		fmt.Println("Installed Versions:")
		for _, v := range versions {
			status := "Not Current"
			if v.IsCurrent {
				status = "Current"
			}
			fmt.Printf("Version: %s, Status: %s\n", v.Version, status)
		}
	}

	a.logToFile(fmt.Sprintf("Found %d installed versions", len(versions)))
	a.logToFile(fmt.Sprintf("Found %d installed versions", versions))
	return versions, nil
}

// GetAvailableNodeVersions attempts to fetch available Node.js versions from the official Node.js API
// If the API fails, it falls back to using the local `nvm` command to retrieve the versions
// GetAvailableNodeVersions 尝试从官方 Node.js API 获取可用的版本信息
// 如果 API 请求失败，则回退到使用本地的 `nvm` 命令来获取版本
func (a *App) GetAvailableNodeVersions() ([]NodeVersionInfo, error) {
	a.logToFile("Fetching available Node.js versions")

	// Attempt to fetch available versions from Node.js API
	// 尝试从 Node.js 官方 API 获取可用版本信息
	resp, err := http.Get("https://nodejs.org/dist/index.json")
	if err == nil && resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			a.logToFile(fmt.Sprintf("Error reading API response: %v", err))
		} else {
			var nodeVersions []NodeAPIResponse
			err = json.Unmarshal(body, &nodeVersions)
			if err != nil {
				a.logToFile(fmt.Sprintf("Error parsing JSON response: %v", err))
			} else {
				// Process API data to generate available versions
				// 处理 API 数据以生成可用版本列表
				var versions []NodeVersionInfo
				installedVersions, err := a.GetInstalledNodeVersions()
				if err != nil {
					a.logToFile(fmt.Sprintf("Error fetching installed versions: %s", err))
					return nil, fmt.Errorf("Error fetching installed versions: %s", err)
				}
				installedMap := make(map[string]bool)
				for _, installed := range installedVersions {
					installedMap[installed.Version] = true
				}

				for _, versionInfo := range nodeVersions {
					status := "Not Installed"
					if installedMap[versionInfo.Version] {
						status = "Installed"
					}

					// 处理 LTS 字段：可以是 bool 或字符串
					// ltsValue := "No"
					// switch v := versionInfo.LTS.(type) {
					// case bool:
					// 	if v {
					// 		ltsValue = "Yes"
					// 	}
					// case string:
					// 	ltsValue = v
					// }

					versions = append(versions, NodeVersionInfo{
						Version:    versionInfo.Version,
						Status:     status,
						NpmVersion: versionInfo.Npm, // 新增字段，将 npm 版本信息添加到结果中
					})

					// a.logToFile(fmt.Sprintf("Version: %s, Status: %s, LTS: %s, NPM: %s", versionInfo.Version, status, ltsValue, versionInfo.Npm))
				}

				a.logToFile(fmt.Sprintf("Found %d available versions from Node.js API", len(versions)))
				return versions, nil
			}
		}
	} else {
		a.logToFile(fmt.Sprintf("Failed to fetch versions from Node.js API: %v", err))
	}

	// Fallback to using nvm command if API fails
	// 如果 API 请求失败，则回退到使用 nvm 命令
	a.logToFile("Falling back to local nvm command to fetch available versions")
	output, err := a.executeNvmCommand("ls", "available")
	if err != nil {
		a.logToFile(fmt.Sprintf("Error fetching available versions: %v", err))
		return nil, fmt.Errorf("Error fetching available versions: %v", err)
	}

	lines := strings.Split(string(output), "\n")
	var versions []NodeVersionInfo
	versionRegex := regexp.MustCompile(`\b\d+\.\d+\.\d+\b`)

	installedVersions, err := a.GetInstalledNodeVersions()
	if err != nil {
		a.logToFile(fmt.Sprintf("Error fetching installed versions: %s", err))
		return nil, fmt.Errorf("Error fetching installed versions: %s", err)
	}
	installedMap := make(map[string]bool)
	for _, installed := range installedVersions {
		installedMap[installed.Version] = true
	}

	// Extract available versions using regex and update their installation status
	// 使用正则表达式提取可用版本并更新其安装状态
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "CURRENT") || strings.Contains(line, "-") {
			continue
		}

		matches := versionRegex.FindAllString(line, -1)
		for _, version := range matches {
			status := "Not Installed"
			if installedMap[version] {
				status = "Installed"
			}
			versions = append(versions, NodeVersionInfo{
				Version:    version,
				Status:     status,
				NpmVersion: "unknown", // 如果使用 nvm 获取的版本信息，不包含 npm，设置为未知
			})
		}
	}

	if a.debugMode {
		fmt.Println("Parsed Available Versions:")
		for _, v := range versions {
			fmt.Printf("Version: %s, Status: %s, Npm: %s\n", v.Version, v.Status, v.NpmVersion)
		}
	}

	a.logToFile(fmt.Sprintf("Found %d available versions from nvm", len(versions)))
	return versions, nil
}
