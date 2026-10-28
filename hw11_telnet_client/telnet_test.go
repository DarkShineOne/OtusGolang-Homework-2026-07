package main

import (
	"bytes"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTelnetClient(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		l, err := net.Listen("tcp", "127.0.0.1:")
		require.NoError(t, err)
		defer func() { require.NoError(t, l.Close()) }()

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()

			in := &bytes.Buffer{}
			out := &bytes.Buffer{}

			timeout, err := time.ParseDuration("10s")
			require.NoError(t, err)

			client := NewTelnetClient(l.Addr().String(), timeout, io.NopCloser(in), out)
			require.NoError(t, client.Connect())
			defer func() { require.NoError(t, client.Close()) }()

			in.WriteString("hello\n")
			err = client.Send()
			require.NoError(t, err)

			err = client.Receive()
			require.NoError(t, err)
			require.Equal(t, "world\n", out.String())
		}()

		go func() {
			defer wg.Done()

			conn, err := l.Accept()
			require.NoError(t, err)
			require.NotNil(t, conn)
			defer func() { require.NoError(t, conn.Close()) }()

			request := make([]byte, 1024)
			n, err := conn.Read(request)
			require.NoError(t, err)
			require.Equal(t, "hello\n", string(request)[:n])

			n, err = conn.Write([]byte("world\n"))
			require.NoError(t, err)
			require.NotEqual(t, 0, n)
		}()

		wg.Wait()
	})
}

func TestTelnetClientConnectError(t *testing.T) {
	t.Run("connection refused", func(t *testing.T) {
		l, err := net.Listen("tcp", "127.0.0.1:")
		require.NoError(t, err)
		addr := l.Addr().String()
		require.NoError(t, l.Close())

		client := NewTelnetClient(addr, time.Second, io.NopCloser(&bytes.Buffer{}), io.Discard)
		require.Error(t, client.Connect())
	})

	t.Run("dial timeout", func(t *testing.T) {
		// TEST-NET-1 address, not routable in practice.
		client := NewTelnetClient("192.0.2.1:4242", 100*time.Millisecond, io.NopCloser(&bytes.Buffer{}), io.Discard)
		require.Error(t, client.Connect())
	})
}

func TestTelnetClientSendStdinEOF(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:")
	require.NoError(t, err)
	defer func() { require.NoError(t, l.Close()) }()

	client := NewTelnetClient(l.Addr().String(), 10*time.Second, io.NopCloser(&bytes.Buffer{}), io.Discard)
	require.NoError(t, client.Connect())
	defer func() { require.NoError(t, client.Close()) }()

	require.NoError(t, client.Send())
}

func TestTelnetClientPeerClosed(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:")
	require.NoError(t, err)
	defer func() { require.NoError(t, l.Close()) }()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		conn, err := l.Accept()
		require.NoError(t, err)

		_, err = conn.Write([]byte("bye\n"))
		require.NoError(t, err)
		require.NoError(t, conn.Close())
	}()

	out := &bytes.Buffer{}
	in := &bytes.Buffer{}

	client := NewTelnetClient(l.Addr().String(), 10*time.Second, io.NopCloser(in), out)
	require.NoError(t, client.Connect())
	defer func() { require.NoError(t, client.Close()) }()

	require.NoError(t, client.Receive())
	require.Equal(t, "bye\n", out.String())

	var sendErr error
	for i := 0; i < 50 && sendErr == nil; i++ {
		in.WriteString("ping\n")
		sendErr = client.Send()
		time.Sleep(20 * time.Millisecond)
	}
	require.Error(t, sendErr)

	wg.Wait()
}

func TestTelnetClientCloseIdempotent(t *testing.T) {
	client := NewTelnetClient("127.0.0.1:4242", time.Second, io.NopCloser(&bytes.Buffer{}), io.Discard)
	require.NoError(t, client.Close())
	require.NoError(t, client.Close())
}
