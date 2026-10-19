package search

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
)

func TestVectorLiteral(t *testing.T) {
	cases := []struct {
		name string
		in   []float32
		want string
	}{
		{"empty", []float32{}, "[]"},
		{"single", []float32{0.5}, "[0.5]"},
		{"multiple", []float32{1.0, -0.25, 0.75}, "[1,-0.25,0.75]"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := vectorLiteral(tc.in)
			if got != tc.want {
				t.Fatalf("vectorLiteral(%v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestFuseRRF(t *testing.T) {
	now := time.Now()
	idA := uuid.New()
	idB := uuid.New()
	idC := uuid.New()

	rowA := sqlc.SearchReportsRow{Type: "lost", ID: idA, Code: "LST-A", ItemName: "Item A", Relevance: 0.8, CreatedAt: now}
	rowB := sqlc.SearchReportsRow{Type: "lost", ID: idB, Code: "LST-B", ItemName: "Item B", Relevance: 0.7, CreatedAt: now}
	rowC := sqlc.SearchReportsRow{Type: "lost", ID: idC, Code: "LST-C", ItemName: "Item C", Relevance: 0.9, CreatedAt: now}

	// Keyword ranking: A (rank 1), B (rank 2)
	kwRows := []sqlc.SearchReportsRow{rowA, rowB}
	// Semantic ranking: B (rank 1), C (rank 2)
	semRows := []sqlc.SearchReportsRow{rowB, rowC}

	// Fused ranking: B is present in both lists, so its RRF score is 1/(60+2) + 1/(60+1) ~= 0.0325
	// A is in kw only: 1/(60+1) ~= 0.0164
	// C is in sem only: 1/(60+2) ~= 0.0161
	// Order should be: B, A, C
	fused := fuseRRF(kwRows, semRows, 0, 10)
	if len(fused) != 3 {
		t.Fatalf("fused count = %d, want 3", len(fused))
	}
	if fused[0].ID != idB {
		t.Fatalf("top item = %s, want B (%s)", fused[0].Code, idB)
	}
	if fused[1].ID != idA {
		t.Fatalf("second item = %s, want A (%s)", fused[1].Code, idA)
	}
	if fused[2].ID != idC {
		t.Fatalf("third item = %s, want C (%s)", fused[2].Code, idC)
	}

	// Test pagination
	page1 := fuseRRF(kwRows, semRows, 0, 2)
	if len(page1) != 3 { // limit + 1 = 3 to indicate next page exists
		t.Fatalf("page1 count = %d, want 3", len(page1))
	}
	page2 := fuseRRF(kwRows, semRows, 2, 2)
	if len(page2) != 1 || page2[0].ID != idC {
		t.Fatalf("page2 unexpected: %+v", page2)
	}
}
