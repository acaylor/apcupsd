// Package apcupsd provides a client for the apcupsd Network Information Server
// (NIS).
package apcupsd

import (
	"context"
	"io"
	"net"
	"sync"
)

// Client is a client for the apcupsd Network Information Server (NIS).
//
// Concurrent calls to Status are serialized. Close may be called at any time,
// including from another goroutine to abort a Status call that is blocked
// waiting on the NIS.
type Client struct {
	mu  sync.Mutex // serializes requests
	rwc io.ReadWriteCloser
}

// Dial dials a connection to an NIS using the address on the named network, and
// creates a Client with the connection.
//
// Typically, network will be one of: "tcp", "tcp4", or "tcp6".
func Dial(network, addr string) (*Client, error) {
	return DialContext(context.Background(), network, addr)
}

// DialContext takes a context and dials a connection to an NIS using the
// address on the named network, and creates a Client with the connection.
//
// The provided Context must be non-nil. If the context expires before the
// connection is complete, an error is returned. Once successfully connected,
// any expiration of the context will not affect the connection.
//
// Typically, network will be one of: "tcp", "tcp4", or "tcp6".
func DialContext(ctx context.Context, network, addr string) (*Client, error) {
	var d net.Dialer
	c, err := d.DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}

	return New(c), nil
}

// New wraps an existing io.ReadWriteCloser to create a Client for communication
// with an NIS. Client's Close method will close the io.ReadWriteCloser when
// called. To allow Close to abort a blocked Status call, closing rwc must
// unblock any pending Read or Write, as it does for a net.Conn.
func New(rwc io.ReadWriteCloser) *Client {
	return &Client{rwc: rwc}
}

// Close closes the connection to an NIS.
func (c *Client) Close() error { return c.rwc.Close() }

const (
	// maxString is the maximum string length for a NIS key/value pair. Value
	// copied from apcupsd source code, v3.14.14.
	maxString = 256
)

// Status retrieves the current UPS status from the NIS.
//
// Status has no timeout of its own; to bound a request, close the Client from
// another goroutine, e.g. with time.AfterFunc.
func (c *Client) Status() (*Status, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := writeFrame(c.rwc, []byte("status")); err != nil {
		return nil, err
	}

	var (
		b    = make([]byte, maxString)
		s    = new(Status)
		perr error
	)

	// NIS server sends text lines containing key/value pairs, so must keep
	// iterating until the zero-length terminator to parse them all.
	for {
		var err error
		b, err = readFrame(c.rwc, b)
		if err != nil {
			return nil, err
		}
		if len(b) == 0 {
			break
		}

		// Keep reading after a parse error so the whole response is consumed
		// and the connection can be reused; report the first error.
		if err := s.parseKV(string(b)); err != nil && perr == nil {
			perr = err
		}
	}

	if perr != nil {
		return nil, perr
	}

	return s, nil
}
