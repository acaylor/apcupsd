package apcupsd

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestClientNoKnownKeyValuePairs(t *testing.T) {
	c := testClient(t, func() [][]byte {
		lenb, kvb := kvBytes("FOO : BAR")
		return [][]byte{
			lenb,
			kvb,
		}
	})

	s, err := c.Status()
	if err != nil {
		t.Fatalf("failed to retrieve status: %v", err)
	}

	if diff := cmp.Diff(&Status{}, s); diff != "" {
		t.Fatalf("unexpected Status (-want +got):\n%s", diff)
	}
}

func TestClientAllTypesKeyValuePairs(t *testing.T) {
	kvs := []string{
		"DATE     : 2016-09-06 22:13:28 -0400",
		"HOSTNAME : example",
		"LOADPCT  :  13.0 Percent Load Capacity",
		"TIMELEFT :  46.5 Minutes",
		"TONBATT  : 0 seconds",
		"NUMXFERS : 0",
		"SELFTEST : NO",
		"NOMPOWER : 865 Watts",
	}

	edt := time.FixedZone("EDT", -60*60*4)
	want := &Status{
		Date:            time.Date(2016, time.September, 6, 22, 13, 28, 0, edt),
		Hostname:        "example",
		LoadPercent:     13.0,
		TimeLeft:        46*time.Minute + 30*time.Second,
		TimeOnBattery:   0 * time.Second,
		NumberTransfers: 0,
		Selftest:        false,
		NominalPower:    865,
	}

	c := testClient(t, func() [][]byte {
		var out [][]byte
		for _, kv := range kvs {
			lenb, kvb := kvBytes(kv)
			out = append(out, lenb)
			out = append(out, kvb)
		}

		return out
	})

	got, err := c.Status()
	if err != nil {
		t.Fatalf("failed to retrieve status: %v", err)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("unexpected Status (-want +got):\n%s", diff)
	}
}

func TestClientTimeout(t *testing.T) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to start listener: %v", err)
	}
	defer l.Close()

	// Force an immediate timeout via context.
	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()

	_, err = DialContext(ctx, "tcp", l.Addr().String())

	var nerr net.Error
	if !errors.As(err, &nerr) || !nerr.Timeout() {
		t.Fatalf("expected timeout error, but got: %v", err)
	}
}

func TestClientLargeFrame(t *testing.T) {
	host := strings.Repeat("x", maxString+1)
	c := New(newTestConn(frames("HOSTNAME : "+host, "")))

	s, err := c.Status()
	if err != nil {
		t.Fatalf("failed to retrieve status: %v", err)
	}
	if s.Hostname != host {
		t.Fatalf("unexpected hostname length: %d", len(s.Hostname))
	}
}

func TestClientUnexpectedEOF(t *testing.T) {
	tests := []struct {
		desc string
		in   []byte
	}{
		{desc: "empty", in: nil},
		{desc: "no terminator", in: frames("HOSTNAME : example")},
		{desc: "length without data", in: []byte{0, 5}},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			_, err := New(newTestConn(tt.in)).Status()
			if !errors.Is(err, io.ErrUnexpectedEOF) {
				t.Fatalf("expected unexpected EOF, but got: %v", err)
			}
		})
	}
}

func TestClientParseErrorConsumesResponse(t *testing.T) {
	c := New(newTestConn(append(
		frames("LINEV : bad", "HOSTNAME : old", ""),
		frames("HOSTNAME : new", "")...,
	)))

	if _, err := c.Status(); err == nil {
		t.Fatal("expected parse error, but got none")
	}

	s, err := c.Status()
	if err != nil {
		t.Fatalf("failed to retrieve status: %v", err)
	}
	if diff := cmp.Diff("new", s.Hostname); diff != "" {
		t.Fatalf("unexpected hostname (-want +got):\n%s", diff)
	}
}

func TestClientCloseUnblocksStatus(t *testing.T) {
	client, server := net.Pipe()
	defer server.Close()

	// Consume the request, then never respond.
	go func() { _, _ = io.Copy(io.Discard, server) }()

	c := New(client)
	errc := make(chan error, 1)
	go func() {
		_, err := c.Status()
		errc <- err
	}()

	// Give Status time to block on the response before closing.
	time.Sleep(50 * time.Millisecond)

	closed := make(chan error, 1)
	go func() { closed <- c.Close() }()

	select {
	case err := <-closed:
		if err != nil {
			t.Fatalf("failed to close: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Close blocked on pending Status")
	}

	select {
	case err := <-errc:
		if err == nil {
			t.Fatal("expected Status error after Close, but got none")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Status did not return after Close")
	}
}

func testClient(t *testing.T, fn func() [][]byte) *Client {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to start listener: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		c, err := l.Accept()
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				return
			}

			panicf("failed to accept connection: %v", err)
		}

		in := make([]byte, 128)
		n, err := c.Read(in)
		if err != nil {
			panicf("failed to read from connection: %v", err)
		}

		status := []byte{0, 6, 's', 't', 'a', 't', 'u', 's'}
		if diff := cmp.Diff(status, in[:n]); diff != "" {
			panicf("unexpected Client request (-want +got):\n%s", diff)
		}

		// Run against test function and append EOF to end of output bytes.
		out := fn()
		out = append(out, []byte{0, 0})

		for _, o := range out {
			if _, err := c.Write(o); err != nil {
				panicf("failed to write to connection: %v", err)
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	c, err := DialContext(ctx, "tcp", l.Addr().String())
	if err != nil {
		t.Fatalf("failed to dial Client: %v", err)
	}

	t.Cleanup(func() {
		cancel()
		wg.Wait()
		_ = c.Close()
		_ = l.Close()
	})

	return c
}

// kvBytes is a helper to generate length and key/value byte buffers.
func kvBytes(kv string) ([]byte, []byte) {
	lenb := make([]byte, 2)
	binary.BigEndian.PutUint16(lenb, uint16(len(kv)))

	return lenb, []byte(kv)
}

func panicf(format string, a ...interface{}) {
	panic(fmt.Sprintf(format, a...))
}
