package procmoni

import (
	"flag"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"syscall"
)

const (
	defaultNumChild = 1
)

var (
	ChildProcFlagName = "pm_child_proc"
)

func init() {
	flag.Bool(ChildProcFlagName, false, "procmoni internal flag")
}

type ChildProcess func()

type ForkChildProcess func() (pid int, err error)

type ParentProcOption struct {
	Logger        Log
	NumChildProc  int
	ChildProcFunc ChildProcess
	ForkChildProc ForkChildProcess
	isChildProc   bool
}

type ParentProcess struct {
	Log
	opt ParentProcOption
}

func NewParentProcess(opt ParentProcOption) *ParentProcess {
	return &ParentProcess{
		opt: opt,
	}
}

func (p *ParentProcess) Run() (err error) {
	p.handleOption()

	if p.opt.isChildProc {
		p.opt.ChildProcFunc()
		return
	}

	os.Args = append(os.Args, "-"+ChildProcFlagName)

	var pid int
	pidList := make([]int, 0, p.opt.NumChildProc)
	pidChangeChan := make([]chan bool, 0, p.opt.NumChildProc)

	for i := 0; i < p.opt.NumChildProc; i++ {
		pid, err = p.opt.ForkChildProc()
		if err != nil {
			p.kill(pidList)
			return
		}
		pidList = append(pidList, pid)
		c := make(chan bool, 1)
		pidChangeChan = append(pidChangeChan, c)
		go p.monitorChildProcState(pid, c)
	}

	p.interceptSign(pidList, pidChangeChan)
	return
}

func (p *ParentProcess) handleOption() {
	if p.opt.Logger == nil {
		p.Log = NewStdLog(logLevelDebug, true)
	} else {
		p.Log = p.opt.Logger
	}
	if p.opt.NumChildProc <= 0 {
		p.opt.NumChildProc = defaultNumChild
	}
	if p.opt.ChildProcFunc == nil {
		p.opt.ChildProcFunc = func() {}
	}
	if p.opt.ForkChildProc == nil {
		p.opt.ForkChildProc = p.defaultForkChildProc
	}
	p.opt.isChildProc = p.isChildProcess()
}

func (p *ParentProcess) isChildProcess() bool {
	isChild := false
	args := make([]string, 0, len(os.Args)+1)
	args = append(args, os.Args[0])
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "-"+ChildProcFlagName) ||
			strings.HasPrefix(arg, "--"+ChildProcFlagName) {
			isChild = true
			continue
		}
		args = append(args, arg)
	}
	os.Args = args // 如果os.Args带了父进程专用的flag而没有过滤，子进程解析flag时可能会报错

	return isChild
}

func (p *ParentProcess) defaultForkChildProc() (pid int, err error) {
	attr := &syscall.ProcAttr{
		Env:   os.Environ(),
		Files: []uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd()},
	}
	return syscall.ForkExec(os.Args[0], os.Args, attr)
}

func (p *ParentProcess) kill(pidList []int) {
	for _, pid := range pidList {
		process, err := os.FindProcess(pid)
		if err != nil {
			p.Errorf("kill(): cann't find process(pid:%d)", pid)
			continue
		}
		err = process.Kill()
		if err != nil {
			p.Errorf("kill(): kill process(pid:%d) err: %s", pid, err)
		}
		p.Infof("kill(): kill process(pid:%d)", pid)
	}
}

func (p *ParentProcess) monitorChildProcState(pid int, c chan bool) {
	defer func() {
		c <- true
	}()

	process, err := os.FindProcess(pid)
	if err != nil {
		p.Errorf("monitor(): cann't find process(pid:%d)", pid)
		return
	}
	state, err := process.Wait()
	if err != nil {
		p.Errorf("monitor(): cann't get process(pid:%d) state", pid)
		return
	}

	p.Infof("monitor(): state of process(pid:%d): %s", pid, state.String())
}

func (p *ParentProcess) interceptSign(pidList []int, pidChangeChan []chan bool) {
	p.Debugf("interceptSign(): pidList: %v", pidList)

	selectCase := make([]reflect.SelectCase, len(pidList)+1)

	for i, _ := range pidList {
		selectCase[i].Dir = reflect.SelectRecv
		selectCase[i].Chan = reflect.ValueOf(pidChangeChan[i])
	}

	signChan := make(chan os.Signal, 1)
	selectCase[len(selectCase)-1].Dir = reflect.SelectRecv
	selectCase[len(selectCase)-1].Chan = reflect.ValueOf(signChan)

	signal.Notify(signChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP, syscall.SIGQUIT)

	var deletePid = func(index int) {
		p.Infof("interceptSign(): stop monitor process(pid:%d)", pidList[index])
		selectCase = append(selectCase[:index], selectCase[index+1:]...)
		pidList = append(pidList[:index], pidList[index+1:]...)
		pidChangeChan = append(pidChangeChan[:index], pidChangeChan[index+1:]...)
	}

	for {
		i, recv, ok := reflect.Select(selectCase)
		if !ok { // close?
			deletePid(i)
			continue
		}
		if i < len(pidList) {
			// 子进程结束了，重新启动一个
			oldPid := pidList[i]
			newPid, err := p.opt.ForkChildProc()
			if err != nil {
				p.Errorf("interceptSign(): old process(pid:%d) closed, fork new process err: %s", oldPid, err)
				deletePid(i)
				continue
			}

			p.Infof("interceptSign(): old process(pid:%d) cloese, monitor new process(pid:%d)", oldPid, newPid)
			pidList[i] = newPid
			go p.monitorChildProcState(newPid, pidChangeChan[i])
			continue
		}

		if i == len(selectCase)-1 { // 系统信号
			// TODO
			p.Debugf("interceptSign(): receive sign: %T", recv.Interface())
			p.kill(pidList)
			break
		}
	}

	p.Info("interceptSign(): break")
}
