package memory

import (
	"testing"
)

func TestVectorConvert(t *testing.T) {
	slice := []float32{1.0, -2.5, 3.14}
	bytes := Float32SliceToBytes(slice)
	restored := BytesToFloat32Slice(bytes)

	if len(restored) != len(slice) {
		t.Fatalf("expected len %d, got %d", len(slice), len(restored))
	}
	for i := range slice {
		if slice[i] != restored[i] {
			t.Errorf("expected %f, got %f", slice[i], restored[i])
		}
	}
}

func TestCosineSimilarity(t *testing.T) {
	v1 := []float32{1.0, 0.0, 0.0}
	v2 := []float32{1.0, 0.0, 0.0}
	sim1 := CosineSimilarity(v1, v2)
	if sim1 < 0.99 {
		t.Errorf("expected identity to be ~1.0, got %f", sim1)
	}

	v3 := []float32{0.0, 1.0, 0.0}
	sim2 := CosineSimilarity(v1, v3)
	if sim2 > 0.01 {
		t.Errorf("expected orthogonal vectors similarity to be ~0.0, got %f", sim2)
	}
}
