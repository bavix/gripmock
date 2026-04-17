package faker

import (
	"crypto/rand"
	"encoding/binary"
	"sync/atomic"
	"time"

	"github.com/brianvoe/gofakeit/v7"
)

//nolint:gochecknoglobals
var seedCounter atomic.Uint64

//nolint:gochecknoinits
func init() {
	seed := randomSeed()
	if seed == 0 {
		seed = 1
	}

	seedCounter.Store(seed)
}

//nolint:ireturn
func New() Generator {
	return NewWithSeed(seedCounter.Add(1))
}

// NewWithSeed returns deterministic generator. Same seed => same sequence.
//
//nolint:ireturn
func NewWithSeed(seed uint64) Generator {
	f := gofakeit.New(seed)
	f.Locked = true

	g := &generator{faker: f}
	g.person.faker = f
	g.contact.faker = f
	g.geo.faker = f
	g.network.faker = f
	g.company.faker = f
	g.commerce.faker = f
	g.text.faker = f
	g.datetime.faker = f
	g.identity.faker = f

	return g
}

func randomSeed() uint64 {
	var b [8]byte
	if _, err := rand.Read(b[:]); err == nil {
		return binary.LittleEndian.Uint64(b[:])
	}

	return uint64(time.Now().UnixNano())
}
