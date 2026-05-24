package onboarding

import "testing"

func TestNumericFromFloat(t *testing.T) {
	n := numericFromFloat(1)
	if !n.Valid {
		t.Fatal("expected valid numeric")
	}
	f, err := n.Float64Value()
	if err != nil || !f.Valid {
		t.Fatalf("Float64Value() = %v, %v", f, err)
	}
	if f.Float64 != 1 {
		t.Fatalf("float = %v, want 1", f.Float64)
	}
}
