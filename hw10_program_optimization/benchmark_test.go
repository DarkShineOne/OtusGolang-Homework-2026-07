//go:build bench
// +build bench

package hw10programoptimization

import (
	"archive/zip"
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func BenchmarkGetDomainStat(b *testing.B) {
	r, err := zip.OpenReader("testdata/users.dat.zip")
	require.NoError(b, err)
	defer r.Close()

	require.Equal(b, 1, len(r.File))

	file, err := r.File[0].Open()
	require.NoError(b, err)

	rawData, err := io.ReadAll(file)
	require.NoError(b, err)
	file.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(rawData)

		stat, err := GetDomainStat(reader, "biz")
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}

		if len(stat) == 0 {
			b.Fatal("expected non-empty stat")
		}
	}
}
