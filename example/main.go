package main

import "flag"

var method string

func init() {
	flag.StringVar(&method, "method", "normal", "run method")
}

func main() {
	flag.Parse()

	switch method {
	case "normal":
		RunNormal()
	}
}
