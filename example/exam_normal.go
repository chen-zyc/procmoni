package main

import (
	"fmt"
	"os"
	"procmoni"
	"time"
)

func RunNormal() {
	childProcess := func() {
		ticker := time.NewTicker(10 * time.Second)
		for range ticker.C {
			fmt.Printf("pid-%d: time: %s\n", os.Getpid(), time.Now().Format("2006-01-02 15:04:05"))
		}
	}
	p := procmoni.NewParentProcess(procmoni.ParentProcOption{
		NumChildProc:  2,
		ChildProcFunc: childProcess,
	})
	err := p.Run()
	if err != nil {
		fmt.Printf("pid-%d: run err: %s", os.Getpid(), err)
	}
}
