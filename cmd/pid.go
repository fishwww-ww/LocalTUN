package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var errInvalidPID = errors.New("invalid pid file")

type pidInfo struct {
	PID        int       `json:"pid"`
	Executable string    `json:"executable,omitempty"`
	StartedAt  time.Time `json:"started_at,omitempty"`
}

func readPID(pidFile string) (int, error) {
	info, err := readPIDInfo(pidFile)
	if err != nil {
		return 0, err
	}
	return info.PID, nil
}

func readPIDInfo(pidFile string) (*pidInfo, error) {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return nil, err
	}

	var info pidInfo
	if err := json.Unmarshal(data, &info); err == nil {
		if info.PID <= 0 {
			return nil, errInvalidPID
		}
		return &info, nil
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return nil, errInvalidPID
	}
	return &pidInfo{PID: pid}, nil
}

func writePIDInfo(pidFile string, pid int) error {
	exe, _ := os.Executable()
	return writePIDInfoWithExecutable(pidFile, pid, filepath.Base(exe))
}

func writePIDInfoWithExecutable(pidFile string, pid int, executable string) error {
	info := pidInfo{
		PID:        pid,
		Executable: executable,
		StartedAt:  time.Now(),
	}
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化 PID 文件失败: %w", err)
	}
	data = append(data, '\n')
	return os.WriteFile(pidFile, data, 0644)
}

func removePIDInfo(pidFile string, pid int) {
	info, err := readPIDInfo(pidFile)
	if err != nil || info.PID != pid {
		return
	}
	_ = os.Remove(pidFile)
}

func processRunning(pid int) bool {
	return processInfoRunning(&pidInfo{PID: pid})
}

func processInfoRunning(info *pidInfo) bool {
	proc, err := os.FindProcess(info.PID)
	if err != nil {
		return false
	}
	if proc.Signal(syscall.Signal(0)) != nil {
		return false
	}
	if info.Executable == "" {
		return true
	}
	name, err := processCommandName(info.PID)
	if err != nil {
		return true
	}
	return name == info.Executable
}

func isRunning(pidFile string) bool {
	info, err := readPIDInfo(pidFile)
	if err != nil {
		return false
	}
	return processInfoRunning(info)
}

func processCommandName(pid int) (string, error) {
	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "comm=").Output()
	if err != nil {
		return "", err
	}
	name := strings.TrimSpace(string(out))
	if name == "" {
		return "", errInvalidPID
	}
	return filepath.Base(name), nil
}
