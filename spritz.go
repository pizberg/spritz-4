// Package spritz provides a pure Go implementation of the Spritz stream cipher
// and hash.
package spritz

import (
	"crypto/cipher"
	"hash"
)

// NewStream returns a new instance of the Spritz cipher using the given key.
func NewStream(key []byte) cipher.Stream {
	var s state
	s.initialize(256)

	// convert to ints
	k := make([]int, len(key))
	for i, v := range key {
		k[i] = int(v)
	}
	s.keySetup(k)

	return stream{s: &s}
}

// NewHash returns a new instance of the Spritz hash with the given output size.
func NewHash(size int) hash.Hash {
	var s state
	d := digest{size: size, s: &s}
	d.Reset()
	return d
}

type stream struct {
	s *state
}

func (s stream) XORKeyStream(dst, src []byte) {
	for i, v := range src {
		dst[i] = v ^ byte(s.s.drip())
	}
}

type digest struct {
	size int
	s    *state
}

func (d digest) Sum(b []byte) []byte {
	s := *d.s // make a local copy
	s.absorbStop()
	s.absorb([]int{d.size})

	out := make([]int, d.size)
	s.squeeze(out)

	h := make([]byte, len(out))
	for i, v := range out {
		h[i] = byte(v)
	}

	return append(b, h...)
}

func (d digest) Write(p []byte) (int, error) {
	msg := make([]int, len(p))
	for i, v := range p {
		msg[i] = int(v)
	}
	d.s.absorb(msg)
	return len(p), nil
}

func (d digest) Size() int {
	return d.size
}

func (d digest) Reset() {
	d.s.initialize(256)
}

func (digest) BlockSize() int {
	return 1 // single byte
}

type state struct {
	// these are all ints instead of bytes to allow for states > 256
	n                int
	s                []int
	a, i, j, k, w, z int
}

func (s *state) initialize(n int) {
	*s = state{
		s: make([]int, 256),
		w: 1,
		n: 256,
	}
	for i := range s.s {
		s.s[i] = i
	}
}

func (s *state) keySetup(key []int) {
	s.absorb(key)
	if s.a > 0 {
		s.shuffle()
	}
}

func (s *state) update() {
	s.i = (s.i + s.w) % s.n
	y := (s.j + s.s[s.i]) % s.n
	s.j = (s.k + s.s[y]) % s.n
	s.k = (s.i + s.k + s.s[s.j]) % s.n
	t := s.s[s.i]
	s.s[s.i] = s.s[s.j]
	s.s[s.j] = t
}

func (s *state) output() int {
	y1 := (s.z + s.k) % s.n
	x1 := (s.i + s.s[y1]) % s.n
	y2 := (s.j + s.s[x1]) % s.n
	s.z = s.s[y2]
	return s.z
}

func (s *state) crush() {
	for i := 0; i < s.n/2; i++ {
		y := (s.n - 1) - i
		x1 := s.s[i]
		x2 := s.s[y]
		if x1 > x2 {
			s.s[i] = x2
			s.s[y] = x1
		} else {
			s.s[i] = x1
			s.s[y] = x2
		}
	}
}

func (s *state) whip() {
	r := s.n * 2
	for i := 0; i < r; i++ {
		s.update()
	}
	s.w = (s.w + 2) % s.n
}

func (s *state) shuffle() {
	s.whip()
	s.crush()
	s.whip()
	s.crush()
	s.whip()
	s.a = 0
}

func (s *state) absorbStop() {
	if s.a == s.n/2 {
		s.shuffle()
	}
	s.a = (s.a + 1) % s.n
}

func (s *state) absorbNibble(x int) {
	if s.a == s.n/2 {
		s.shuffle()
	}
	y := (s.n/2 + x) % s.n
	t := s.s[s.a]
	s.s[s.a] = s.s[y]
	s.s[y] = t
	s.a = (s.a + 1) % s.n
}

func (s *state) absorbValue(b int) {
	d := s.n / 16
	s.absorbNibble(b % d) // LOW
	s.absorbNibble(b / d) // HIGH
}

func (s *state) absorb(msg []int) {
	for _, v := range msg {
		s.absorbValue(v)
	}
}

func (s *state) drip() int {
	if s.a > 0 {
		s.shuffle()
	}
	s.update()
	return s.output()
}

func (s *state) squeeze(out []int) {
	if s.a > 0 {
		s.shuffle()
	}
	for i := range out {
		out[i] = s.drip()
	}
}