package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const defaultTimeout = 10 * time.Second

func main() {
	var timeout time.Duration

	flag.DurationVar(&timeout, "timeout", defaultTimeout, "connection timeout")
	flag.Parse()

	if flag.NArg() != 2 {
		fmt.Fprintf(os.Stderr, "usage: go-telnet [--timeout=%s] host port\n", defaultTimeout)
		os.Exit(1)
	}

	addr := net.JoinHostPort(flag.Arg(0), flag.Arg(1))

	client := NewTelnetClient(addr, timeout, os.Stdin, os.Stdout)
	if err := client.Connect(); err != nil {
		fmt.Fprintf(os.Stderr, "...%v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "...Connected to %s\n", addr)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var shutdown sync.Once

	notify := func(msg string) {
		shutdown.Do(func() {
			fmt.Fprint(os.Stderr, msg)
			stop()
		})
	}

	recvDone := make(chan struct{})

	go func() {
		if err := client.Send(); err != nil {
			if !errors.Is(err, net.ErrClosed) {
				notify("...Connection was closed by peer\n")
			}
			return
		}

		notify("...EOF\n")
	}()

	go func() {
		defer close(recvDone)

		if err := client.Receive(); err != nil {
			if !errors.Is(err, net.ErrClosed) {
				notify(fmt.Sprintf("...Receive failed: %v\n", err))
			}
			return
		}

		notify("...Connection was closed by peer\n")
	}()

	<-ctx.Done()

	if err := client.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "...%v\n", err)
	}

	// Горутину Send намеренно не ждём: терминал Linux должен корректно закрывать программу
	<-recvDone
}
