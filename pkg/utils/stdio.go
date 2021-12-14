package utils

import (
	"net"
	"os"
	"time"
)

type Stdio struct{}

// Read implements io.Reader interface.
func (Stdio) Read(b []byte) (int, error) { return os.Stdin.Read(b) }

// Write implements io.Writer interface.
func (Stdio) Write(b []byte) (int, error) { return os.Stdout.Write(b) }

// Close implements io.Closer interface.
func (Stdio) Close() error {
	if err := os.Stdin.Close(); err != nil {
		return err
	}
	return os.Stdout.Close()
}

// LocalAddr implements net.Conn interface.
func (s Stdio) LocalAddr() net.Addr { return s }

// RemoteAddr implements net.Conn interface.
func (s Stdio) RemoteAddr() net.Addr { return s }

// SetDeadline implements net.Conn interface.
func (Stdio) SetDeadline(t time.Time) error { return nil }

// SetReadDeadline implements net.Conn interface.
func (Stdio) SetReadDeadline(t time.Time) error { return nil }

// SetWriteDeadline implements net.Conn interface.
func (Stdio) SetWriteDeadline(t time.Time) error { return nil }

// Network implements net.Addr interface.
func (Stdio) Network() string { return "Stdio" }

// String implements net.Addr interface.
func (Stdio) String() string { return "Stdio" }
