package bloom

import (
	"hash/fnv"
	"math"
	"sync"
)

// Filter is a probabilistic set membership structure.
type Filter struct {
	mu   sync.RWMutex
	bits []uint64
	m    uint64
	k    int
	n    int
}

// New creates a bloom filter sized for n expected items and fpRate false-positive rate.
func New(n int, fpRate float64) *Filter {
	if n < 1 {
		n = 1
	}
	if fpRate <= 0 || fpRate >= 1 {
		fpRate = 0.01
	}
	m := uint64(math.Ceil(-float64(n) * math.Log(fpRate) / (math.Ln2 * math.Ln2)))
	if m < 64 {
		m = 64
	}
	k := int(math.Ceil((float64(m) / float64(n)) * math.Ln2))
	if k < 1 {
		k = 1
	}
	if k > 30 {
		k = 30
	}
	words := (m + 63) / 64
	return &Filter{
		bits: make([]uint64, words),
		m:    m,
		k:    k,
	}
}

// Add inserts a key into the filter.
func (f *Filter) Add(key string) {
	if f == nil {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, idx := range f.indexes(key) {
		word, bit := idx/64, idx%64
		f.bits[word] |= 1 << bit
	}
	f.n++
}

// MightContain reports whether key may be in the set (false positives possible).
func (f *Filter) MightContain(key string) bool {
	if f == nil {
		return false
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	for _, idx := range f.indexes(key) {
		word, bit := idx/64, idx%64
		if f.bits[word]&(1<<bit) == 0 {
			return false
		}
	}
	return true
}

// Len returns approximate inserted count.
func (f *Filter) Len() int {
	if f == nil {
		return 0
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.n
}

// Reset clears the filter.
func (f *Filter) Reset() {
	if f == nil {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := range f.bits {
		f.bits[i] = 0
	}
	f.n = 0
}

func (f *Filter) indexes(key string) []uint64 {
	h1 := hash64(key, 0x9e3779b97f4a7c15)
	h2 := hash64(key, 0xc2b2ae3d27d4eb4f)
	if h2 == 0 {
		h2 = 0x9e3779b97f4a7c15
	}
	out := make([]uint64, f.k)
	k := f.k
	if k < 0 {
		k = 0
	}
	for i := 0; i < k; i++ {
		out[i] = (h1 + uint64(i)*h2) % f.m // #nosec G115 -- i is non-negative and bounded by k
	}
	return out
}

func hash64(s string, seed uint64) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte{byte(seed), byte(seed >> 8), byte(seed >> 16), byte(seed >> 24)})
	_, _ = h.Write([]byte(s))
	return h.Sum64()
}
