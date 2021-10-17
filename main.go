package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/jdbaldry/go-language-server-protocol/jsonrpc2"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
)

type stdio struct{}

// Read implements io.Reader interface.
func (stdio) Read(b []byte) (int, error) { return os.Stdin.Read(b) }

// Write implements io.Writer interface.
func (stdio) Write(b []byte) (int, error) { return os.Stdout.Write(b) }

// Close implements io.Closer interface.
func (stdio) Close() error {
	if err := os.Stdin.Close(); err != nil {
		return err
	}
	return os.Stdout.Close()
}

// LocalAddr implements net.Conn interface.
func (s stdio) LocalAddr() net.Addr { return s }

// RemoteAddr implements net.Conn interface.
func (s stdio) RemoteAddr() net.Addr { return s }

// SetDeadline implements net.Conn interface.
func (stdio) SetDeadline(t time.Time) error { return nil }

// SetReadDeadline implements net.Conn interface.
func (stdio) SetReadDeadline(t time.Time) error { return nil }

// SetWriteDeadline implements net.Conn interface.
func (stdio) SetWriteDeadline(t time.Time) error { return nil }

// Network implements net.Addr interface.
func (stdio) Network() string { return "stdio" }

// String implements net.Addr interface.
func (stdio) String() string { return "stdio" }

func main() {
	ctx := context.TODO()
	stream := jsonrpc2.NewHeaderStream(stdio{})
	conn := jsonrpc2.NewConn(stream)
	client := protocol.ClientDispatcher(conn)

	s, err := newServer(client)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	conn.Go(ctx, protocol.Handlers(
		protocol.ServerHandler(s, jsonrpc2.MethodNotFound)))
	<-conn.Done()
	if err := conn.Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
