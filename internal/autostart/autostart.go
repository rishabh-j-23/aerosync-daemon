package autostart

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Enable sets up the application to run automatically on login
func Enable() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	absExe, _ := filepath.Abs(exe)

	switch runtime.GOOS {
	case "windows":
		return enableWindows(absExe)
	case "linux":
		return enableLinux(absExe)
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// Disable removes the automatic startup configuration
func Disable() error {
	switch runtime.GOOS {
	case "windows":
		return disableWindows()
	case "linux":
		return disableLinux()
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// IsEnabled checks if auto-start is currently configured
func IsEnabled() bool {
	switch runtime.GOOS {
	case "windows":
		return isEnabledWindows()
	case "linux":
		return isEnabledLinux()
	default:
		return false
	}
}

func getStartupPath() string {
	appData := os.Getenv("APPDATA")
	return filepath.Join(appData, "Microsoft", "Windows", "Start Menu", "Programs", "Startup", "aerosync_autostart.bat")
}

func enableWindows(exePath string) error {
	path := getStartupPath()
	// Using timeout gives the system 10 seconds to 'settle' after login before Aerosync starts.
	// This ensures we don't fight for resources during the critical login window.
	// 'start' then spawns the process and the batch script exits immediately.
	content := fmt.Sprintf("@echo off\ntimeout /t 10 /nobreak > NUL\nstart \"\" \"%s\" start", exePath)
	return os.WriteFile(path, []byte(content), 0644)
}

func disableWindows() error {
	path := getStartupPath()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	// Also attempt to clean up the old Task Scheduler task if it exists (legacy)
	exec.Command("schtasks", "/Delete", "/TN", "Aerosync", "/F").Run()
	return nil
}

func isEnabledWindows() bool {
	path := getStartupPath()
	_, err := os.Stat(path)
	return err == nil
}

func enableLinux(exePath string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	unitDir := filepath.Join(home, ".config", "systemd", "user")
	if err := os.MkdirAll(unitDir, 0755); err != nil {
		return err
	}

	content := fmt.Sprintf(`[Unit]
Description=Aerosync Background Service
After=network.target

[Service]
ExecStart=%s start
Restart=on-failure
RestartSec=30

[Install]
WantedBy=default.target
`, exePath)

	unitPath := filepath.Join(unitDir, "aerosync.service")
	if err := os.WriteFile(unitPath, []byte(content), 0644); err != nil {
		return err
	}

	exec.Command("systemctl", "--user", "daemon-reload").Run()
	if output, err := exec.Command("systemctl", "--user", "enable", "--now", "aerosync.service").CombinedOutput(); err != nil {
		return fmt.Errorf("failed to enable systemd service: %v (Output: %s)", err, string(output))
	}

	return nil
}

func disableLinux() error {
	exec.Command("systemctl", "--user", "disable", "--now", "aerosync.service").Run()
	
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	unitPath := filepath.Join(home, ".config", "systemd", "user", "aerosync.service")
	return os.Remove(unitPath)
}

func isEnabledLinux() bool {
	cmd := exec.Command("systemctl", "--user", "is-enabled", "aerosync.service")
	err := cmd.Run()
	return err == nil
}
