package isureal

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"strconv"
	"testing"
)

func BenchmarkComputeImageSignature(b *testing.B) {
	k := bytes.Repeat([]byte{1}, 32)
	for b.Loop() {
		computeImageSignature(k, "0192f1c2-1111-7000-8000-000000000000", 1761900000)
	}
}

func TestComputeImageSignatureMatchesNaive(t *testing.T) {
	for _, k := range [][]byte{bytes.Repeat([]byte{1}, 32), bytes.Repeat([]byte{0xab}, 32)} {
		for _, exp := range []int64{100, 1761900000} {
			id := "0192f1c2-1111-7000-8000-000000000000"
			value := []byte(strconv.FormatInt(exp, 10) + ":" + id)
			for range imageSignatureIterations {
				mac := hmac.New(sha256.New, k)
				mac.Write(value)
				value = mac.Sum(nil)
			}
			if !bytes.Equal(value, computeImageSignature(k, id, exp)) {
				t.Fatal("signature changed")
			}
		}
	}
}
