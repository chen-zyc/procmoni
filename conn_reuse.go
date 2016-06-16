package procmoni

import (
	"net"
	"os"
	"sync"
	"syscall"
)

func ConnReuse(addr string) (p ForkChildProcess) {
	var once sync.Once
	var fd uintptr
	var initErr error

	var init = func() {
		var listener net.Listener
		var file *os.File

		listener, initErr = net.Listen("tcp", addr)
		if initErr != nil {
			return
		}
		file, initErr = listener.(*net.TCPListener).File()
		if initErr != nil {
			return
		}

		fd = file.Fd()
	}

	p = func() (pid int, err error) {
		once.Do(init)

		if initErr != nil {
			err = initErr
			return
		}

		attr := &syscall.ProcAttr{
			Env:   os.Environ(),
			Files: []uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd(), fd},
		}
		return syscall.ForkExec(os.Args[0], os.Args, attr)
	}
	return
}
