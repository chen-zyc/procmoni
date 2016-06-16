package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"procmoni"
)

func RunReuseConn() {
	childProcess := func() {
		f := os.NewFile(3, "")
		listener, err := net.FileListener(f)
		if err != nil {
			log.Fatal(err)
		}

		http.Serve(listener, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fmt.Sprintf("pid#%d: hello", os.Getpid())))
		}))
	}

	p := procmoni.NewParentProcess(procmoni.ParentProcOption{
		NumChildProc:  2,
		ChildProcFunc: childProcess,
		ForkChildProc: procmoni.ConnReuse(":8080"),
	})

	err := p.Run()
	if err != nil {
		fmt.Printf("pid-%d: run err: %s", os.Getpid(), err)
	}
}
