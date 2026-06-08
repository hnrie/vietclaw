package memory

import (
	"encoding/binary"
	"math"
)

func Float32SliceToBytes(slice []float32) []byte {
	buf := make([]byte, len(slice)*4)
	for i, f := range slice {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(f))
	}
	return buf
}

func BytesToFloat32Slice(buf []byte) []float32 {
	if len(buf)%4 != 0 {
		return nil
	}
	slice := make([]float32, len(buf)/4)
	for i := 0; i < len(slice); i++ {
		slice[i] = math.Float32frombits(binary.LittleEndian.Uint32(buf[i*4:]))
	}
	return slice
}

func CosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0.0
	}
	var dotProduct float64
	var normA float64
	var normB float64

	for i := range a {
		av := float64(a[i])
		bv := float64(b[i])
		dotProduct += av * bv
		normA += av * av
		normB += bv * bv
	}

	if normA == 0.0 || normB == 0.0 {
		return 0.0
	}
	return float32(dotProduct / (math.Sqrt(normA) * math.Sqrt(normB)))
}
