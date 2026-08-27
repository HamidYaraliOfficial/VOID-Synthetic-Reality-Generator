// Package wsutil implements a minimal RFC 6455 WebSocket server endpoint
// using only the Go standard library (net/http hijacking + crypto/sha1),
// avoiding an external dependency such as gorilla/websocket. It supports
// exactly what VOID's real-time metrics/event stream needs: text-frame
// send, and close — which is sufficient for a one-way server-push feed.
package wsutil

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
)

const magicGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// Conn is an upgraded WebSocket connection.
type Conn struct {
	rw   *bufio.ReadWriter
	mu   sync.Mutex
	closed bool
}

// Upgrade performs the HTTP -> WebSocket handshake (RFC 6455 section 4) on a
// hijackable ResponseWriter and returns a Conn ready for WriteText/Close.
func Upgrade(w http.ResponseWriter, r *http.Request) (*Conn, error) {
	if r.Header.Get("Upgrade") != "websocket" {
		return nil, errors.New("wsutil: not a websocket upgrade request")
	}
	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		return nil, errors.New("wsutil: missing Sec-WebSocket-Key")
	}
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		return nil, errors.New("wsutil: ResponseWriter does not support hijacking")
	}
	conn, rw, err := hijacker.Hijack()
	if err != nil {
		return nil, err
	}

	accept := computeAccept(key)
	resp := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + accept + "\r\n\r\n"
	if _, err := rw.WriteString(resp); err != nil {
		conn.Close()
		return nil, err
	}
	if err := rw.Flush(); err != nil {
		conn.Close()
		return nil, err
	}
	return &Conn{rw: rw}, nil
}

func computeAccept(key string) string {
	h := sha1.New()
	h.Write([]byte(key + magicGUID))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// WriteText sends one text frame (server frames are never masked, per spec).
func (c *Conn) WriteText(payload []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return errors.New("wsutil: connection closed")
	}
	if err := writeFrame(c.rw, 0x1, payload); err != nil {
		return err
	}
	return c.rw.Flush()
}

// ReadMessage blocks for the next client frame, unmasking per spec, and
// returns its payload. Used to receive simple client control messages
// (e.g. {"action":"unsubscribe"}); ping/pong/close opcodes are handled
// transparently.
func (c *Conn) ReadMessage() ([]byte, error) {
	for {
		opcode, payload, err := readFrame(c.rw)
		if err != nil {
			return nil, err
		}
		switch opcode {
		case 0x8: // close
			c.Close()
			return nil, io.EOF
		case 0x9: // ping -> pong
			c.mu.Lock()
			_ = writeFrame(c.rw, 0xA, payload)
			_ = c.rw.Flush()
			c.mu.Unlock()
			continue
		case 0x1, 0x2:
			return payload, nil
		default:
			continue
		}
	}
}

func (c *Conn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	_ = writeFrame(c.rw, 0x8, nil)
	_ = c.rw.Flush()
	return nil
}

func writeFrame(rw *bufio.ReadWriter, opcode byte, payload []byte) error {
	header := []byte{0x80 | opcode} // FIN=1
	l := len(payload)
	switch {
	case l <= 125:
		header = append(header, byte(l))
	case l <= 65535:
		header = append(header, 126)
		buf := make([]byte, 2)
		binary.BigEndian.PutUint16(buf, uint16(l))
		header = append(header, buf...)
	default:
		header = append(header, 127)
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, uint64(l))
		header = append(header, buf...)
	}
	if _, err := rw.Write(header); err != nil {
		return err
	}
	_, err := rw.Write(payload)
	return err
}

func readFrame(rw *bufio.ReadWriter) (opcode byte, payload []byte, err error) {
	first, err := rw.ReadByte()
	if err != nil {
		return 0, nil, err
	}
	opcode = first & 0x0f
	second, err := rw.ReadByte()
	if err != nil {
		return 0, nil, err
	}
	masked := second&0x80 != 0
	length := int64(second & 0x7f)
	switch length {
	case 126:
		buf := make([]byte, 2)
		if _, err := io.ReadFull(rw, buf); err != nil {
			return 0, nil, err
		}
		length = int64(binary.BigEndian.Uint16(buf))
	case 127:
		buf := make([]byte, 8)
		if _, err := io.ReadFull(rw, buf); err != nil {
			return 0, nil, err
		}
		length = int64(binary.BigEndian.Uint64(buf))
	}
	var maskKey [4]byte
	if masked {
		if _, err := io.ReadFull(rw, maskKey[:]); err != nil {
			return 0, nil, err
		}
	}
	data := make([]byte, length)
	if _, err := io.ReadFull(rw, data); err != nil {
		return 0, nil, err
	}
	if masked {
		for i := range data {
			data[i] ^= maskKey[i%4]
		}
	}
	return opcode, data, nil
}

// SafeUpgradeCheckOrigin is a small helper handlers can call before Upgrade
// to reject cross-origin WebSocket handshakes from unexpected hosts.
func SafeUpgradeCheckOrigin(r *http.Request, allowed []string) bool {
	origin := r.Header.Get("Origin")
	if origin == "" || len(allowed) == 0 {
		return true
	}
	for _, a := range allowed {
		if a == "*" || a == origin {
			return true
		}
	}
	return false
}

var _ = fmt.Sprintf // keep fmt import if unused paths trimmed later
