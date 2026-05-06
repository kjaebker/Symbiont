package enums

import "testing"

func TestSetHas(t *testing.T) {
	s := NewSet("a", "b", "c")
	if !s.Has("b") {
		t.Error("expected Has(b) to be true")
	}
	if s.Has("d") {
		t.Error("expected Has(d) to be false")
	}
}

func TestSetValuesSorted(t *testing.T) {
	s := NewSet("c", "a", "b", "a")
	got := s.Values()
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("len mismatch: got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("values[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestSetValuesIndependent(t *testing.T) {
	// Mutating a returned slice must not affect future calls.
	s := NewSet("a", "b")
	v := s.Values()
	v[0] = "z"
	if got := s.Values(); got[0] != "a" {
		t.Errorf("returned slice mutated set: got %v", got)
	}
}
