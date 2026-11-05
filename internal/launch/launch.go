package launch

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

const signalExitBase = 128

type Process struct {
	cmd *exec.Cmd
}

func Start(argv, env []string) (*Process, error) {
	if len(argv) == 0 {
		return nil, errors.New("no program to launch")
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("launch %s: %w", argv[0], err)
	}
	return &Process{cmd: cmd}, nil
}

func (p *Process) Signal(sig os.Signal) {
	_ = p.cmd.Process.Signal(sig)
}

func (p *Process) Wait() (int, error) {
	err := p.cmd.Wait()
	var exitErr *exec.ExitError
	if err != nil && !errors.As(err, &exitErr) {
		return 0, fmt.Errorf("wait for program: %w", err)
	}
	status, ok := p.cmd.ProcessState.Sys().(syscall.WaitStatus)
	if ok && status.Signaled() {
		return signalExitBase + int(status.Signal()), nil
	}
	return p.cmd.ProcessState.ExitCode(), nil
}
