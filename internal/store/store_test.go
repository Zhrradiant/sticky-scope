package store_test

import (
	"testing"

	"sticky-scope/internal/store"
)

func TestCASDedupRoundTripAndGC(t *testing.T) {
	st, err := store.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	h1, err := st.PutBytes([]byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	h2, err := st.PutBytes([]byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if h1 != h2 {
		t.Fatalf("identical content produced different hashes: %s vs %s", h1, h2)
	}
	if !st.Has(h1) {
		t.Fatal("Has returned false right after Put")
	}
	got, err := st.GetBytes(h1)
	if err != nil || string(got) != "hello" {
		t.Fatalf("round-trip failed: %q err=%v", got, err)
	}

	h3, err := st.PutBytes([]byte("world"))
	if err != nil {
		t.Fatal(err)
	}
	// Reference only h1; GC must reclaim h3.
	deleted, err := st.GC(map[string]struct{}{h1: {}})
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Fatalf("GC deleted %d blobs, want 1", deleted)
	}
	if st.Has(h3) {
		t.Error("unreferenced blob survived GC")
	}
	if !st.Has(h1) {
		t.Error("referenced blob was deleted by GC")
	}
}