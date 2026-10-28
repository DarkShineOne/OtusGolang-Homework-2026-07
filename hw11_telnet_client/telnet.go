package main

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// TelnetClient — минимальный telnet-клиент без поддержки протокола, опций и команд.
type TelnetClient interface {
	Connect() error
	io.Closer
	Send() error
	Receive() error
}

// NewTelnetClient создаёт TelnetClient, который читает из in, пишет в out и работает по адресу address.
func NewTelnetClient(address string, timeout time.Duration, in io.ReadCloser, out io.Writer) TelnetClient {
	return &telnetClient{
		address: address,
		timeout: timeout,
		in:      in,
		out:     out,
	}
}

type telnetClient struct {
	address string
	timeout time.Duration
	in      io.ReadCloser
	out     io.Writer

	once sync.Once
	conn net.Conn
}

// Connect устанавливает TCP-соединение с адресом в пределах таймаута.
func (c *telnetClient) Connect() error {
	conn, err := net.DialTimeout("tcp", c.address, c.timeout)
	if err != nil {
		return fmt.Errorf("dial %s: %w", c.address, err)
	}

	c.conn = conn

	return nil
}

// Send копирует полное сообщение из входного потока в соединение.
// nil-возврат означает, что входной поток достиг EOF (Ctrl+D).
func (c *telnetClient) Send() error {
	_, err := io.Copy(c.conn, c.in)

	return err
}

// Receive копирует полное сообщение из соединения в выходной поток.
// nil-возврат означает, что соединение закрыто второй стороной.
func (c *telnetClient) Receive() error {
	_, err := io.Copy(c.out, c.conn)

	return err
}

// Close освобождает соединение и входной поток; повторные вызовы ничего не делают.
func (c *telnetClient) Close() error {
	var firstErr error

	c.once.Do(func() {
		if c.conn != nil {
			firstErr = c.conn.Close()
		}

		if err := c.in.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	})

	return firstErr
}
