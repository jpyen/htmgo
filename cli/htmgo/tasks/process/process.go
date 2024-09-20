package process

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"
)

type CmdWithFlags struct {
	flags []RunFlag
	cmd   *exec.Cmd
}

var workingDir string
var commands = make([]CmdWithFlags, 0)

func AppendRunning(cmd *exec.Cmd, flags ...RunFlag) {
	slog.Debug("running", slog.String("command", strings.Join(cmd.Args, " ")),
		slog.String("dir", cmd.Dir),
		slog.String("cwd", GetWorkingDir()))
	commands = append(commands, CmdWithFlags{flags: flags, cmd: cmd})
}

func GetWorkingDir() string {
	if workingDir == "" {
		wd, _ := os.Getwd()
		return wd
	}
	return workingDir
}

func SetWorkingDir(dir string) {
	workingDir = dir
}

func GetPathRelativeToCwd(path string) string {
	return filepath.Join(GetWorkingDir(), path)
}

func shouldSkipKilling(flags []RunFlag, skipFlag []RunFlag) bool {
	for _, flag := range flags {
		if slices.Contains(skipFlag, flag) {
			return true
		}
	}
	return false
}

func KillAll(skipFlag ...RunFlag) {

	tries := 0
	for {
		tries++
		allFinished := true
		for _, cmd := range commands {
			if cmd.cmd.Process == nil {
				allFinished = false

				if tries > 50 {
					args := strings.Join(cmd.cmd.Args, " ")
					slog.Debug("process %v is not running after 50 tries, breaking.\n", args)
					allFinished = true
					break
				} else {
					time.Sleep(time.Millisecond * 50)
					continue
				}
			}
		}
		if allFinished {
			break
		}
	}

	for _, command := range commands {
		if shouldSkipKilling(command.flags, skipFlag) {
			continue
		}
		pid := command.cmd.Process.Pid
		err := syscall.Kill(-pid, syscall.SIGKILL)
		if err != nil {
			continue
		}
	}

	for {
		finished := true
		for _, c := range commands {
			if c.cmd.Process == nil {
				continue
			}
			if shouldSkipKilling(c.flags, skipFlag) {
				continue
			}
			exists, err := PidExists(int32(c.cmd.Process.Pid))
			if err != nil {
				finished = false
			}
			if exists {
				syscall.Kill(-c.cmd.Process.Pid, syscall.SIGKILL)
				finished = false
			}
		}

		if finished {
			break
		} else {
			slog.Debug("waiting for all processes to exit\n")
			time.Sleep(time.Millisecond * 5)
		}
	}

	commands = make([]CmdWithFlags, 0)
	slog.Debug("all processes killed\n")
}

func PidExists(pid int32) (bool, error) {
	if pid <= 0 {
		return false, fmt.Errorf("invalid pid %v", pid)
	}
	proc, err := os.FindProcess(int(pid))
	if err != nil {
		return false, err
	}
	err = proc.Signal(syscall.Signal(0))
	if err == nil {
		return true, nil
	}
	if err.Error() == "os: process already finished" {
		return false, nil
	}
	var errno syscall.Errno
	ok := errors.As(err, &errno)
	if !ok {
		return false, err
	}
	switch errno {
	case syscall.ESRCH:
		return false, nil
	case syscall.EPERM:
		return true, nil
	}
	return false, err
}

func RunOrExit(command string) {
	_ = Run(command, ExitOnError)
}

type RunFlag int

const (
	ExitOnError RunFlag = iota
	Silent
	KillOnlyOnExit
)

func RunMany(commands []string, flags ...RunFlag) error {
	for _, command := range commands {
		err := Run(command, flags...)
		if err != nil {
			if slices.Contains(flags, ExitOnError) {
				os.Exit(1)
			}
			return err
		}
	}
	return nil
}

func Run(command string, flags ...RunFlag) error {
	parts := strings.Fields(command)
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if slices.Contains(flags, Silent) {
		cmd.Stdout = nil
		cmd.Stderr = nil
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if workingDir != "" {
		cmd.Dir = workingDir
	}

	AppendRunning(cmd, flags...)

	err := cmd.Run()

	if err == nil {
		return nil
	}

	if strings.Contains(err.Error(), "signal: killed") {
		return nil
	}

	if slices.Contains(flags, ExitOnError) {
		slog.Error("Error running command: ",
			slog.String("error", err.Error()),
			slog.String("command", command))
		os.Exit(1)
	}

	return err
}