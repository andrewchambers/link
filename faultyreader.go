package link

import (
	"io"
	"math/rand"
)

type faultyReader struct {
	errPoint uint32
	rc       io.ReadCloser
}

func NewFaultyReader(byteErrorRate float64, rc io.ReadCloser) io.ReadCloser {

	errPoint := uint32(byteErrorRate * float64(0xffffffff))

	return &faultyReader{errPoint, rc}
}

func (fr *faultyReader) Read(b []byte) (int, error) {
	n, err := fr.rc.Read(b)
	for i := 0; i < n; i++ {
		if rand.Uint32() <= fr.errPoint {
			b[i] = byte(int(rand.Uint32()))
		}
	}
	return n, err
}

func (fr *faultyReader) Close() error {
	return fr.rc.Close()
}
