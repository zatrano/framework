package http

import (
	cryptorand "crypto/rand"
	"encoding/binary"
	mrand "math/rand/v2"
	"sync"
	"time"
)

// idSources are ChaCha8 generators seeded once from crypto/rand.
// A request id is a correlation token. It stays unguessable without a
// getrandom syscall on every request.
var idSources = sync.Pool{
	New: func() any {
		var seed [32]byte
		if _, err := cryptorand.Read(seed[:]); err != nil {
			binary.LittleEndian.PutUint64(seed[:8], uint64(time.Now().UnixNano()))
		}
		return mrand.New(mrand.NewChaCha8(seed))
	},
}

// GenerateRequestID writes 32 hex digits into the request.
// The digits stay in idBuf. No string is allocated.
func (r *Request) GenerateRequestID() {
	if r == nil {
		return
	}
	src := idSources.Get().(*mrand.Rand)
	var raw [16]byte
	fillID(src, raw[:])
	idSources.Put(src)
	encodeHex32(r.idBuf[:], raw[:])
	r.idN = 32
	r.requestID = ""
}

func fillID(src *mrand.Rand, b []byte) {
	var n uint64
	for i := range b {
		if i%8 == 0 {
			n = src.Uint64()
		}
		b[i] = byte(n)
		n >>= 8
	}
}

func encodeHex32(dst, src []byte) {
	const digits = "0123456789abcdef"
	for i := 0; i < 16; i++ {
		b := src[i]
		dst[i*2] = digits[b>>4]
		dst[i*2+1] = digits[b&0x0f]
	}
}

// UseRequestID copies the request's correlation id onto the response.
// A generated id is copied as bytes. An incoming id is the string already
// parsed from the request header.
func (r *Response) UseRequestID(req *Request) {
	if r == nil || req == nil {
		return
	}
	if req.requestID != "" {
		r.Header("X-Request-Id", req.requestID)
		return
	}
	if req.idN != 32 {
		return
	}
	copy(r.reqID[:], req.idBuf[:])
	r.reqIDOn = true
	r.WithoutHeader("X-Request-Id")
}
