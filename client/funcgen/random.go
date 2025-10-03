package main

import (
	crand "crypto/rand"
	"encoding/binary"
	"fmt"
	"math/rand/v2"
)

// A global seed that influences all RNGs equally.
var GlobalSeed uint64 = 0

// Return a deteministic random number source based on the PCG algorithm.
// https://go.dev/blog/chacha8rand#performance
func NewRand(seed uint64) rand.Source {
	return rand.NewPCG(GlobalSeed, seed)
}

// Use the seed to get one deterministic random number and
// then return another rand source at an offset from that.
func NewOffsetRand(seed, offset uint64) rand.Source {
	r := NewRand(seed).Uint64()
	return NewRand(r + offset)
}

// Return an actually random number from the system's cryptographic randomness.
// This is not deterministic, obviuosly.
func TrueRandom() uint64 {
	var b [8]byte
	_, err := crand.Read(b[:])
	if err != nil {
		panic(fmt.Sprintf("cannot read system randomness: %v", err))
	}
	return binary.LittleEndian.Uint64(b[:])
}
