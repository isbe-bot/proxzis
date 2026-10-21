package house

import (
	"testing"

	"github.com/google/uuid"
)

func TestRosterLoadsSheet(t *testing.T) {
	agents := Agents()
	if len(agents) != 10 {
		t.Fatalf("agents %d", len(agents))
	}
	whites := agents[0]
	if whites.Slug != "the-tall-whites" || whites.Name != "TheTallWhites" || whites.ID != uuid.MustParse("b0000000-0000-0000-0000-000000000001") {
		t.Fatalf("%+v", whites)
	}
	if whites.Rock != 25 || whites.Paper != 40 || whites.Scissors != 35 {
		t.Fatalf("tactics rock=%d paper=%d scissors=%d", whites.Rock, whites.Paper, whites.Scissors)
	}
	if Pick(1500).Slug != "ms-cloud" {
		t.Fatalf("closest to 1500 is %s", Pick(1500).Slug)
	}
	if Pick(1670).Slug != "aileun" {
		t.Fatalf("closest to 1670 is %s", Pick(1670).Slug)
	}
}
