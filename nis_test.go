package apcupsd

import (
	"bytes"
	"errors"
	"io"
	"math"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_readFrame(t *testing.T) {
	r := bytes.NewReader(frames("HELLO : WORLD", ""))

	b, err := readFrame(r, nil)
	if err != nil {
		t.Fatalf("failed to read frame: %v", err)
	}
	if diff := cmp.Diff("HELLO : WORLD", string(b)); diff != "" {
		t.Fatalf("unexpected frame (-want +got):\n%s", diff)
	}

	b, err = readFrame(r, b)
	if err != nil {
		t.Fatalf("failed to read terminator: %v", err)
	}
	if len(b) != 0 {
		t.Fatalf("expected empty terminator frame, but got: %q", b)
	}
}

func Test_readFrameLarge(t *testing.T) {
	// Larger than the buffer passed in, which must grow rather than panic.
	in := strings.Repeat("x", maxString+1)

	b, err := readFrame(bytes.NewReader(frames(in)), make([]byte, maxString))
	if err != nil {
		t.Fatalf("failed to read frame: %v", err)
	}
	if diff := cmp.Diff(in, string(b)); diff != "" {
		t.Fatalf("unexpected frame (-want +got):\n%s", diff)
	}
}

func Test_readFrameUnexpectedEOF(t *testing.T) {
	tests := []struct {
		desc string
		in   []byte
	}{
		{desc: "no length", in: nil},
		{desc: "partial length", in: []byte{0}},
		{desc: "no data", in: []byte{0, 5}},
		{desc: "partial data", in: []byte{0, 5, 'a'}},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			_, err := readFrame(bytes.NewReader(tt.in), nil)
			if !errors.Is(err, io.ErrUnexpectedEOF) {
				t.Fatalf("expected unexpected EOF, but got: %v", err)
			}
		})
	}
}

func Test_writeFrameBufferTooLarge(t *testing.T) {
	err := writeFrame(io.Discard, make([]byte, math.MaxUint16+1))
	if !errors.Is(err, errBufferTooLarge) {
		t.Fatalf("expected buffer too large, but got: %v", err)
	}
}

func Test_writeFrame(t *testing.T) {
	var buf bytes.Buffer
	if err := writeFrame(&buf, []byte("status")); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	if diff := cmp.Diff(frames("status"), buf.Bytes()); diff != "" {
		t.Fatalf("unexpected byte output (-want +got):\n%s", diff)
	}
}

// frames encodes each string as an NIS frame. An empty string encodes the
// end-of-response terminator.
func frames(ss ...string) []byte {
	var b []byte
	for _, s := range ss {
		b = append(b, byte(len(s)>>8), byte(len(s)))
		b = append(b, s...)
	}

	return b
}

// testConn is an in-memory io.ReadWriteCloser that serves reads from a fixed
// byte stream and discards writes.
type testConn struct {
	io.Reader
}

func newTestConn(b []byte) *testConn { return &testConn{Reader: bytes.NewReader(b)} }

func (*testConn) Write(b []byte) (int, error) { return len(b), nil }
func (*testConn) Close() error                { return nil }
