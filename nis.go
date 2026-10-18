package apcupsd

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
)

// The NIS protocol frames each message as:
//  - 2 bytes: big endian length of data
//  - N bytes: data
//
// A response is a sequence of frames terminated by a zero-length frame.

// errBufferTooLarge indicates that writeFrame was passed a buffer that is too
// large to send to the NIS.
var errBufferTooLarge = errors.New("apcupsd: buffer too large; must be size of uint16 or less")

// readFrame reads one frame from r into buf, growing buf if needed, and returns
// the frame's data. An empty result is the end-of-response terminator.
func readFrame(r io.Reader, buf []byte) ([]byte, error) {
	var lenb [2]byte
	if err := readFull(r, lenb[:]); err != nil {
		return nil, err
	}

	n := int(binary.BigEndian.Uint16(lenb[:]))
	if n > cap(buf) {
		buf = make([]byte, n)
	}
	buf = buf[:n]

	if err := readFull(r, buf); err != nil {
		return nil, err
	}

	return buf, nil
}

// readFull is io.ReadFull, except that the transport ending before b is filled
// is always io.ErrUnexpectedEOF: a response is only complete once the
// zero-length terminator frame is read.
func readFull(r io.Reader, b []byte) error {
	_, err := io.ReadFull(r, b)
	if err == io.EOF {
		return io.ErrUnexpectedEOF
	}

	return err
}

// writeFrame writes b to w as a single frame.
func writeFrame(w io.Writer, b []byte) error {
	if len(b) > math.MaxUint16 {
		return errBufferTooLarge
	}

	frame := binary.BigEndian.AppendUint16(make([]byte, 0, 2+len(b)), uint16(len(b)))
	_, err := w.Write(append(frame, b...))
	return err
}
