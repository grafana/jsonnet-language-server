package main

import (
	"net"
	"os"
	"time"
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
