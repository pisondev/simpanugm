package cursor

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRoundTrip(t *testing.T) {
	at := time.Date(2026, 11, 2, 7, 15, 0, 123, time.UTC)
	id := uuid.Must(uuid.NewV7())
	k, err := Decode(Encode(at, id))
	if err != nil || !k.At.Equal(at) || k.ID != id {
		t.Fatalf("round trip: %+v %v", k, err)
	}
	n, err := DecodeOffset(EncodeOffset(40))
	if err != nil || n != 40 {
		t.Fatalf("offset: %d %v", n, err)
	}
	bad := "not-a-cursor!"
	if _, err := Decode(&bad); err == nil {
		t.Fatal("garbage accepted")
	}
	if k, err := Decode(nil); k != nil || err != nil {
		t.Fatal("nil cursor is the first page")
	}
}

func TestLimit(t *testing.T) {
	for in, want := range map[int]int{0: 20, -1: 20, 5: 5, 100: 100, 101: 100} {
		v := in
		if got := Limit(&v); got != want {
			t.Errorf("Limit(%d) = %d", in, got)
		}
	}
	if Limit(nil) != 20 {
		t.Fatal("default")
	}
}
