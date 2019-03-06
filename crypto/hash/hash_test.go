package hash

import (
	"github.com/stretchr/testify/require"
	"testing"
)

var someData = []byte("testing")

const (
	ExpectedSha256    = "cf80cd8aed482d5d1527d7dc72fceff84e6326592848447d2dc0b0e87dfc9a90"
	ExpectedKeccak256 = "5f16f4c7f149ac4f9510d9cf8cf384038ad348b3bcdc01915f95de12df9d1b02"
)

func TestCalcSha256(t *testing.T) {
	h := CalcSha256(someData)
	require.Len(t, h, SHA256_HASH_SIZE_BYTES, "Sha256 is in incorrect length")
	require.Equal(t, ExpectedSha256, h.String(), "result should match")
}

func TestCalcSha256_MultipleChunks(t *testing.T) {
	h := CalcSha256(someData[:3], someData[3:])
	require.Len(t, h, SHA256_HASH_SIZE_BYTES, "Sha256 invalid length in multiple chunks")
	require.Equal(t, ExpectedSha256, h.String(), "result should match")
}

func TestCalcKeccak256(t *testing.T) {
	h := CalcKeccak256(someData)
	require.Len(t, h, KECCAK256_HASH_SIZE_BYTES, "Keccak is in invalid length")
	require.Equal(t, ExpectedKeccak256, h.String(), "result should match")
}

func TestCalcKeccak256_MultipleChunks(t *testing.T) {
	h := CalcKeccak256(someData[:3], someData[3:])
	require.Len(t, h, KECCAK256_HASH_SIZE_BYTES, "Keccak invalid length in multiple chunks")
	require.Equal(t, ExpectedKeccak256, h.String(), "result should match")
}

func BenchmarkCalcSha256(b *testing.B) {
	for i := 0; i < b.N; i++ {
		CalcSha256(someData)
	}
}

func BenchmarkCalcKeccak256(b *testing.B) {
	for i := 0; i < b.N; i++ {
		CalcKeccak256(someData)
	}
}
