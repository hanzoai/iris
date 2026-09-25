// Package iris is the screen stream: schema/iris.zap bound to Go (iris_zap.go,
// zapgen) plus the framing zap-proto/go's transport keeps unexported.
//
// A frame is [u32 LE 1+len][dir][body]. Over a WebSocket the length is dropped:
// one binary message = [dir][body]. dir 3 opens (body = rpc request envelope),
// 4 carries a message (body = [u32 stream][ZAP msg, msgType = kind]), 5 ends.
package iris

//go:generate go run github.com/zap-proto/go/cmd/zapgen -single -out . schema/iris.zap

import (
	"encoding/binary"
	"errors"
	"io"

	"github.com/zap-proto/go/rpc"
)

// Directions (zap-proto/go transport).
const (
	Open byte = 3
	Msg  byte = 4
	End  byte = 5
)

// Kinds: ZAP header msgType of a dir-4 message.
const (
	KCredit  uint8 = 1
	KPointer uint8 = 2
	KWheel   uint8 = 3
	KKey     uint8 = 4
	KCommit  uint8 = 5
	KClip    uint8 = 6
	KResize  uint8 = 7
	KRefresh uint8 = 8
	KPick    uint8 = 9
	KFiles   uint8 = 10
	KCdp     uint8 = 11
	KSize    uint8 = 32
	KTile    uint8 = 33
	KCursor  uint8 = 34
	KPoint   uint8 = 35
	KWant    uint8 = 36
	KStats   uint8 = 37
	KBye     uint8 = 38
)

// Mode (Size.Mode).
const (
	Browser uint8 = 1
	Desktop uint8 = 2
)

// Max bounds one frame (a 4K q90 tile is ~2 MiB).
const Max = 8 << 20

var ErrFrame = errors.New("iris: frame out of range")

// Tag stamps kind into a built message's header flags.
func Tag(msg []byte, kind uint8) []byte {
	binary.LittleEndian.PutUint16(msg[6:8], uint16(kind)<<8)
	return msg
}

// Kind reads a message's kind.
func Kind(msg []byte) uint8 {
	if len(msg) < 8 {
		return 0
	}
	return uint8(binary.LittleEndian.Uint16(msg[6:8]) >> 8)
}

// Body is a dir-4 body: stream id then msg.
func Body(stream uint32, msg []byte) []byte {
	b := make([]byte, 4+len(msg))
	binary.LittleEndian.PutUint32(b, stream)
	copy(b[4:], msg)
	return b
}

// Split undoes Body.
func Split(body []byte) (uint32, []byte, bool) {
	if len(body) < 4 {
		return 0, nil, false
	}
	return binary.LittleEndian.Uint32(body), body[4:], true
}

// Hi builds the dir-3 body: Iris.watch(hello) carrying cap.
func Hi(stream uint32, cap, hello []byte) []byte {
	return rpc.BuildRequest(rpc.Call{Method: IrisWatchOrdinal, PromiseID: stream, Cap: cap, Payload: hello})
}

// Read reads one frame.
func Read(r io.Reader) (byte, []byte, error) {
	var h [5]byte
	if _, err := io.ReadFull(r, h[:]); err != nil {
		return 0, nil, err
	}
	n := binary.LittleEndian.Uint32(h[:4])
	if n < 1 || n > Max {
		return 0, nil, ErrFrame
	}
	b := make([]byte, n-1)
	if _, err := io.ReadFull(r, b); err != nil {
		return 0, nil, err
	}
	return h[4], b, nil
}

// Write writes one frame in a single call, so concurrent writers on a
// serialised writer never interleave.
func Write(w io.Writer, dir byte, body []byte) error {
	if 1+len(body) > Max {
		return ErrFrame
	}
	b := make([]byte, 5+len(body))
	binary.LittleEndian.PutUint32(b, uint32(1+len(body)))
	b[4] = dir
	copy(b[5:], body)
	_, err := w.Write(b)
	return err
}
