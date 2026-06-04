package cmd

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"syscall"
)

var errInvalidPID = errors.New("invalid pid file")

func readPID(pidFile string) (int, error) {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return 0, errInvalidPID
	}
	return pid, nil
}

func processRunning(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

func isRunning(pidFile string) bool {
	pid, err := readPID(pidFile)
	if err != nil {
		return false
	}
	return processRunning(pid)
}
