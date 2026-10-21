package parley

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestSpectatorContractMatchesEngine fails until the site uses the engine
// matrix (strike beats feint), a five-round regulation match, and pass-line copy.
func TestSpectatorContractMatchesEngine(t *testing.T) {
	root := moduleRoot(t)
	arena := string(mustRead(t, filepath.Join(root, "web", "src", "utils", "arena.js")))
	seed := string(mustRead(t, filepath.Join(root, "internal", "catalog", "catalog.go")))
	app := string(mustRead(t, filepath.Join(root, "web", "src", "App.vue")))
	var problems []string
	for _, needle := range []string{"strike: 'feint'", "feint: 'parry'", "parry: 'strike'"} {
		if !strings.Contains(arena, needle) {
			problems = append(problems, "arena.js BEATS is not the engine matrix; missing "+needle)
		}
	}
	if strings.Contains(seed, "SpectatorRounds = 7") {
		problems = append(problems, "seeded archive plays 7 rounds; engine regulation is 5, first to 3")
	}
	if strings.Contains(app, "Best-of-7") {
		problems = append(problems, "overview says Best-of-7; engine default is first to 3 within 5 rounds")
	}
	if strings.Contains(app, "Strike <span>beats Parry</span>") {
		problems = append(problems, "overview says Strike beats Parry; engine Strike beats Feint")
	}
	if strings.Contains(app, "winner takes the pot") {
		problems = append(problems, "overview uses a head-to-head pot; engine settlement is a unilateral pass-line against the House")
	}
	if len(problems) > 0 {
		t.Fatalf("spectator data is not the live Parley contract:\n- %s", strings.Join(problems, "\n- "))
	}
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller unavailable")
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
