package launch

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

const signalExitBase = 128

type Process struct {
	cmd  *exec.Cmd
	done chan struct{}
	code int
	err  error
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
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("launch %s: %w", argv[0], err)
	}
	p := &Process{cmd: cmd, done: make(chan struct{})}
	go p.wait()
	return p, nil
}

func (p *Process) Done() <-chan struct{} {
	return p.done
}

func (p *Process) Result() (int, error) {
	<-p.done
	return p.code, p.err
}

func (p *Process) Signal(sig os.Signal) {
	if s, ok := sig.(syscall.Signal); ok {
		_ = syscall.Kill(-p.cmd.Process.Pid, s)
		return
	}
	_ = p.cmd.Process.Signal(sig)
}

func (p *Process) Terminate(grace time.Duration) {
	p.Signal(syscall.SIGTERM)
	select {
	case <-p.done:
		return
	case <-time.After(grace):
	}
	p.Signal(syscall.SIGKILL)
	<-p.done
}

func (p *Process) wait() {
	defer close(p.done)
	err := p.cmd.Wait()
	var exitErr *exec.ExitError
	if err != nil && !errors.As(err, &exitErr) {
		p.err = fmt.Errorf("wait for program: %w", err)
		return
	}
	status, ok := p.cmd.ProcessState.Sys().(syscall.WaitStatus)
	if ok && status.Signaled() {
		p.code = signalExitBase + int(status.Signal())
		return
	}
	p.code = p.cmd.ProcessState.ExitCode()
}
