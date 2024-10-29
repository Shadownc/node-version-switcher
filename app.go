package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
)

// App struct
type App struct {
	ctx         context.Context
	debugMode   bool
	enableLogs  bool
	logFilePath string // 新增字段
}

// NodeVersion 结构体存储已安装版本信息
type NodeVersion struct {
	Version   string
	IsCurrent bool
}

// NodeVersionInfo 结构体存储每个版本的详细信息
type NodeVersionInfo struct {
	Version string
	Status  string
}

// NewApp creates a new App application struct
func NewApp() *App {
	// 获取可执行文件所在目录
	execPath, err := os.Executable()
	logPath := "nvm-switcher.log"
	if err == nil {
		logPath = filepath.Join(filepath.Dir(execPath), "nvm-switcher.log")
	}

	return &App{
		debugMode:   false,
		enableLogs:  false,
		logFilePath: logPath,
	}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.logToFile("Application started")
}

// shutdown is called when the app is closing
func (a *App) shutdown(ctx context.Context) {
	a.logToFile("Application shutting down")
}

// beforeClose is called when the user tries to close the app
func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	if a.enableLogs {
		a.logToFile("Application closing initiated")
	}

	// 如果是通过系统关闭按钮退出，也要清理托盘
	go quitSystray()

	return false
}

// executeNvmCommand 执行 nvm 命令的辅助函数
func (a *App) executeNvmCommand(args ...string) ([]byte, error) {
	cmd := exec.Command("nvm", args...)

	if !a.debugMode {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow: true,
		}
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		a.logToFile(fmt.Sprintf("Command failed: nvm %v\nError: %v\nOutput: %s\n",
			args, err, string(output)))
	}

	return output, err
}

// logToFile 记录日志到文件
func (a *App) logToFile(message string) {
	// 添加调试输出
	fmt.Printf("Debug: enableLogs = %v, debugMode = %v\n", a.enableLogs, a.debugMode)

	if !a.enableLogs {
		fmt.Println("Debug: Logging is disabled, returning")
		return
	}

	// 确保日志目录存在
	logDir := filepath.Dir(a.logFilePath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Printf("Debug: Failed to create log directory: %v\n", err)
		// 如果创建目录失败，尝试使用用户目录
		userHome, err := os.UserHomeDir()
		if err == nil {
			a.logFilePath = filepath.Join(userHome, "nvm-switcher.log")
			logDir = userHome
		} else {
			fmt.Printf("Debug: Failed to get user home directory: %v\n", err)
			return
		}
	}

	fmt.Printf("Debug: Attempting to create/open log file at: %s\n", a.logFilePath)

	f, err := os.OpenFile(a.logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Debug: Failed to open log file: %v\n", err)
		return
	}
	defer f.Close()

	timeStamp := time.Now().Format("2006-01-02 15:04:05")
	logMessage := fmt.Sprintf("[%s] %s\n", timeStamp, message)

	_, writeErr := f.Write([]byte(logMessage))
	if writeErr != nil {
		fmt.Printf("Debug: Failed to write to log file: %v\n", writeErr)
		return
	}

	fmt.Printf("Debug: Successfully wrote log message: %s\n", message)
}

// InstallNodeVersion installs a specific Node.js version using nvm
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

// UninstallNodeVersion uninstalls a specific Node.js version using nvm
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

// SwitchNodeVersion switches the Node.js version using nvm
func (a *App) SwitchNodeVersion(version string) string {
	a.logToFile(fmt.Sprintf("Attempting to switch to Node.js version: %s", version))
	output, err := a.executeNvmCommand("use", version)
	if err != nil {
		errMsg := fmt.Sprintf("Error switching to Node.js %s: %s", version, string(output))
		a.logToFile(errMsg)
		return errMsg
	}

	// 切换成功后可以调用 GetInstalledNodeVersions 重新加载状态
	installedVersions, err := a.GetInstalledNodeVersions()
	if err != nil {
		errMsg := fmt.Sprintf("Switched to Node.js %s, but failed to update status: %s", version, err)
		a.logToFile(errMsg)
		return errMsg
	}

	if a.debugMode {
		fmt.Println("Updated Installed Versions after switching:")
		for _, v := range installedVersions {
			status := "Not Current"
			if v.IsCurrent {
				status = "Current"
			}
			fmt.Printf("Version: %s, Status: %s\n", v.Version, status)
		}
	}

	successMsg := fmt.Sprintf("Successfully switched to Node.js %s", version)
	a.logToFile(successMsg)
	return successMsg
}

// GetAvailableNodeVersions fetches the list of available Node.js versions
func (a *App) GetAvailableNodeVersions() ([]NodeVersionInfo, error) {
	a.logToFile("Fetching available Node.js versions")
	output, err := a.executeNvmCommand("ls", "available")
	if err != nil {
		a.logToFile(fmt.Sprintf("Error fetching available versions: %v", err))
		return nil, fmt.Errorf("Error fetching available versions: %v", err)
	}

	lines := strings.Split(string(output), "\n")
	var versions []NodeVersionInfo
	versionRegex := regexp.MustCompile(`\b\d+\.\d+\.\d+\b`) // 匹配 X.Y.Z 格式的版本号

	// 获取已安装的版本列表
	installedVersions, err := a.GetInstalledNodeVersions()
	if err != nil {
		a.logToFile(fmt.Sprintf("Error fetching installed versions: %s", err))
		return nil, fmt.Errorf("Error fetching installed versions: %s", err)
	}
	installedMap := make(map[string]bool)
	for _, installed := range installedVersions {
		installedMap[installed.Version] = true
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "CURRENT") || strings.Contains(line, "-") {
			continue
		}

		// 查找版本号并将其添加到结果列表中
		matches := versionRegex.FindAllString(line, -1)
		for _, version := range matches {
			status := "Not Installed"
			if installedMap[version] {
				status = "Installed"
			}
			versions = append(versions, NodeVersionInfo{
				Version: version,
				Status:  status,
			})
		}
	}

	if a.debugMode {
		fmt.Println("Parsed Available Versions:")
		for _, v := range versions {
			fmt.Printf("Version: %s, Status: %s\n", v.Version, v.Status)
		}
	}

	a.logToFile(fmt.Sprintf("Found %d available versions", len(versions)))
	return versions, nil
}

// GetInstalledNodeVersions fetches the list of locally installed Node.js versions
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
	return versions, nil
}
