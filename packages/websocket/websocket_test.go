package websocket

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"testing"
)

func TestWriteBinaryPingPongClose(t *testing.T) {
	var out bytes.Buffer
	c := &Conn{bufrw: bufio.NewReadWriter(bufio.NewReader(&bytes.Buffer{}), bufio.NewWriter(&out))}

	if err := c.WriteBinary([]byte{1, 2, 3}); err != nil {
		t.Fatal(err)
	}
	if err := c.Ping([]byte("hi")); err != nil {
		t.Fatal(err)
	}
	if err := c.Pong([]byte("hi")); err != nil {
		t.Fatal(err)
	}
	if err := c.Close(1000); err != nil {
		t.Fatal(err)
	}

	data := out.Bytes()
	op, payload, rest := mustReadUnmaskedFrame(t, data)
	if op != 0x2 || !bytes.Equal(payload, []byte{1, 2, 3}) {
		t.Fatalf("binary op=%#x payload=%v", op, payload)
	}
	op, payload, rest = mustReadUnmaskedFrame(t, rest)
	if op != 0x9 || string(payload) != "hi" {
		t.Fatalf("ping op=%#x payload=%q", op, payload)
	}
	op, payload, rest = mustReadUnmaskedFrame(t, rest)
	if op != 0xA || string(payload) != "hi" {
		t.Fatalf("pong op=%#x payload=%q", op, payload)
	}
	op, payload, rest = mustReadUnmaskedFrame(t, rest)
	if op != 0x8 || len(payload) != 2 || binary.BigEndian.Uint16(payload) != 1000 {
		t.Fatalf("close op=%#x payload=%v", op, payload)
	}
	if len(rest) != 0 {
		t.Fatalf("trailing %v", rest)
	}
}

func TestReadMessagePingAutoPong(t *testing.T) {
	ping := buildMaskedFrame(0x9, []byte("ping-me"))
	in := bytes.NewBuffer(ping)
	var out bytes.Buffer
	c := &Conn{bufrw: bufio.NewReadWriter(bufio.NewReader(in), bufio.NewWriter(&out))}

	op, payload, err := c.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if op != 0x9 || string(payload) != "ping-me" {
		t.Fatalf("op=%#x payload=%q", op, payload)
	}
	op, pongPayload, rest := mustReadUnmaskedFrame(t, out.Bytes())
	if op != 0xA || string(pongPayload) != "ping-me" {
		t.Fatalf("auto pong op=%#x payload=%q", op, pongPayload)
	}
	if len(rest) != 0 {
		t.Fatalf("trailing %v", rest)
	}
}

func TestReadMessageCloseEOF(t *testing.T) {
	frame := buildMaskedFrame(0x8, []byte{0x03, 0xe8})
	c := &Conn{bufrw: bufio.NewReadWriter(bufio.NewReader(bytes.NewReader(frame)), bufio.NewWriter(&bytes.Buffer{}))}
	op, _, err := c.ReadMessage()
	if op != 0x8 || err != io.EOF {
		t.Fatalf("op=%#x err=%v", op, err)
	}
}

func buildMaskedFrame(opcode byte, payload []byte) []byte {
	mask := [4]byte{1, 2, 3, 4}
	header := []byte{0x80 | opcode, byte(0x80 | len(payload))}
	header = append(header, mask[:]...)
	masked := make([]byte, len(payload))
	for i := range payload {
		masked[i] = payload[i] ^ mask[i%4]
	}
	return append(header, masked...)
}

func mustReadUnmaskedFrame(t *testing.T, data []byte) (opcode byte, payload []byte, rest []byte) {
	t.Helper()
	if len(data) < 2 {
		t.Fatalf("short frame %v", data)
	}
	opcode = data[0] & 0x0f
	n := int(data[1] & 0x7f)
	off := 2
	if n == 126 {
		n = int(binary.BigEndian.Uint16(data[2:4]))
		off = 4
	}
	payload = data[off : off+n]
	rest = data[off+n:]
	return opcode, payload, rest
}
