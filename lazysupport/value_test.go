package lazysupport

import "testing"

func TestValue(t *testing.T) {

	out := NewValue([]byte("hello")).SHA256().Base58().String()
	if out != "42TEXg1vFAbcJ65y7qdYG9iCPvYfy3NDdVLd75akX2P5" {
		t.Fatalf("expected %v, got %v", "", out)
	}
}
