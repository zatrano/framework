package websocket

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	stdhttp "net/http"
	"net/url"
	"strings"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/routing"
)

const acceptGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// Handler processes an upgraded websocket connection.
type Handler func(conn *Conn) error

// CheckOrigin decides whether a WebSocket upgrade Origin is allowed.
// Return true to accept. A nil check uses SameOrigin.
type CheckOrigin func(req *http.Request) bool

// Upgrade upgrades matching requests to WebSocket using SameOrigin checks.
func Upgrade(handler Handler) routing.HandlerFunc {
	return UpgradeWithCheckOrigin(handler, nil)
}

// UpgradeWithCheckOrigin upgrades with a custom Origin policy.
// If check is nil, SameOrigin is used. Pass AllowAnyOrigin to disable checks
// (development only — never in production with cookie-authenticated sockets).
func UpgradeWithCheckOrigin(handler Handler, check CheckOrigin) routing.HandlerFunc {
	if check == nil {
		check = SameOrigin
	}
	return func(req *http.Request) *http.Response {
		return http.Hijack(func(w stdhttp.ResponseWriter) error {
			hj, ok := w.(stdhttp.Hijacker)
			if !ok {
				stdhttp.Error(w, "hijacking not supported", stdhttp.StatusInternalServerError)
				return fmt.Errorf("hijacking not supported")
			}
			key := req.Header("Sec-WebSocket-Key")
			if key == "" || !strings.EqualFold(req.Header("Upgrade"), "websocket") {
				stdhttp.Error(w, "expected websocket upgrade", stdhttp.StatusBadRequest)
				return nil
			}
			if !check(req) {
				stdhttp.Error(w, "origin not allowed", stdhttp.StatusForbidden)
				return nil
			}

			conn, bufrw, err := hj.Hijack()
			if err != nil {
				return err
			}
			defer conn.Close()

			accept := acceptKey(key)
			response := "HTTP/1.1 101 Switching Protocols\r\n" +
				"Upgrade: websocket\r\n" +
				"Connection: Upgrade\r\n" +
				"Sec-WebSocket-Accept: " + accept + "\r\n\r\n"
			if _, err := bufrw.WriteString(response); err != nil {
				return err
			}
			if err := bufrw.Flush(); err != nil {
				return err
			}

			ws := &Conn{bufrw: bufrw}
			return handler(ws)
		})
	}
}

// SameOrigin allows missing Origin (non-browser clients) or Origin host matching Request.Host.
func SameOrigin(req *http.Request) bool {
	origin := strings.TrimSpace(req.Header("Origin"))
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	return strings.EqualFold(u.Host, req.Host())
}

// AllowAnyOrigin accepts every Origin (explicit opt-out of CSRF-style WS protection).
func AllowAnyOrigin(req *http.Request) bool {
	return true
}

func acceptKey(key string) string {
	h := sha1.New()
	_, _ = io.WriteString(h, key+acceptGUID)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// Conn is a WebSocket connection supporting text, binary, and control frames.
type Conn struct {
	bufrw *bufio.ReadWriter
}

// ReadMessage reads the next text/binary frame payload.
func (c *Conn) ReadMessage() (opcode byte, payload []byte, err error) {
	header := make([]byte, 2)
	if _, err = io.ReadFull(c.bufrw, header); err != nil {
		return 0, nil, err
	}
	opcode = header[0] & 0x0f
	masked := header[1]&0x80 != 0
	length := int(header[1] & 0x7f)
	if length == 126 {
		ext := make([]byte, 2)
		if _, err = io.ReadFull(c.bufrw, ext); err != nil {
			return 0, nil, err
		}
		length = int(binary.BigEndian.Uint16(ext))
	} else if length == 127 {
		ext := make([]byte, 8)
		if _, err = io.ReadFull(c.bufrw, ext); err != nil {
			return 0, nil, err
		}
		length = int(binary.BigEndian.Uint64(ext))
	}

	var mask [4]byte
	if masked {
		if _, err = io.ReadFull(c.bufrw, mask[:]); err != nil {
			return 0, nil, err
		}
	}
	payload = make([]byte, length)
	if _, err = io.ReadFull(c.bufrw, payload); err != nil {
		return 0, nil, err
	}
	if masked {
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
	}
	if opcode == 0x8 {
		return opcode, payload, io.EOF
	}
	// Auto-reply to ping control frames with a matching pong.
	if opcode == 0x9 {
		if err = c.Pong(payload); err != nil {
			return opcode, payload, err
		}
	}
	return opcode, payload, nil
}

// WriteText writes a text frame.
func (c *Conn) WriteText(message string) error {
	return c.writeFrame(0x1, []byte(message))
}

// WriteBinary writes a binary frame.
func (c *Conn) WriteBinary(data []byte) error {
	return c.writeFrame(0x2, data)
}

// Ping writes a ping control frame.
func (c *Conn) Ping(payload []byte) error {
	return c.writeFrame(0x9, payload)
}

// Pong writes a pong control frame.
func (c *Conn) Pong(payload []byte) error {
	return c.writeFrame(0xA, payload)
}

// Close writes a close control frame. Optional status is a big-endian uint16
// followed by an optional UTF-8 reason (RFC 6455).
func (c *Conn) Close(status ...uint16) error {
	var payload []byte
	if len(status) > 0 {
		payload = make([]byte, 2)
		binary.BigEndian.PutUint16(payload, status[0])
	}
	return c.writeFrame(0x8, payload)
}

func (c *Conn) writeFrame(opcode byte, payload []byte) error {
	header := []byte{0x80 | opcode}
	n := len(payload)
	switch {
	case n < 126:
		header = append(header, byte(n))
	case n <= 65535:
		header = append(header, 126, byte(n>>8), byte(n))
	default:
		ext := make([]byte, 8)
		binary.BigEndian.PutUint64(ext, uint64(n))
		header = append(header, 127)
		header = append(header, ext...)
	}
	if _, err := c.bufrw.Write(header); err != nil {
		return err
	}
	if _, err := c.bufrw.Write(payload); err != nil {
		return err
	}
	return c.bufrw.Flush()
}
