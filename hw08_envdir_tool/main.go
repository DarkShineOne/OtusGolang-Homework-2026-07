package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "usage: %s <env-dir> command [args...]\n", os.Args[0])
		os.Exit(1)
	}

	env, err := ReadDir(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "envdir: %v\n", err)
		os.Exit(1)
	}

	os.Exit(RunCmd(os.Args[2:], env))
}
