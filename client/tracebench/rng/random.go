package rng

import (
	crand "crypto/rand"
	"encoding/binary"
	"fmt"
	"math/rand/v2"
)

// A source of rand.Sources.
type SeededSourcer struct {
	commonSeed uint64
}

func NewSeededSourcer(global uint64) *SeededSourcer {
	return &SeededSourcer{global}
}

// Return a deteministic random number source based on the PCG algorithm,
// using the struct seed as first and seed arg as second half of internal state.
// https://go.dev/blog/chacha8rand#performance
func (s *SeededSourcer) New(seed uint64) rand.Source {
	return rand.NewPCG(s.commonSeed, seed)
}

// Use the seed to get a single deterministic random number and reuse that
// number with the offset to return another deterministic rand source.
func (s *SeededSourcer) NewAtOffset(seed, offset uint64) rand.Source {
	r := s.New(seed).Uint64()
	return s.New(r + offset)
}

// Return an actually random number from the system's cryptographic randomness.
// This is not deterministic, obviously.
func TrueRandom() uint64 {
	var b [8]byte
	_, err := crand.Read(b[:])
	if err != nil {
		panic(fmt.Sprintf("cannot read system randomness: %v", err))
	}
	return binary.LittleEndian.Uint64(b[:])
}
