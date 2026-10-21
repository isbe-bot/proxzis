package parley

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestSpectatorContractMatchesEngine checks the archive uses the engine matrix
// and does not write the settlement ledger. The seeded series is seven rounds
// of sample history, stored apart from live pass-line settlement.
func TestSpectatorContractMatchesEngine(t *testing.T) {
	root := moduleRoot(t)
	arena := string(mustRead(t, filepath.Join(root, "web", "src", "utils", "arena.js")))
	store := string(mustRead(t, filepath.Join(root, "internal", "catalog", "sqlstore.go")))
	app := string(mustRead(t, filepath.Join(root, "web", "src", "App.vue")))
	var problems []string
	for _, needle := range []string{"strike: 'feint'", "feint: 'parry'", "parry: 'strike'"} {
		if !strings.Contains(arena, needle) {
			problems = append(problems, "arena.js BEATS is not the engine matrix; missing "+needle)
		}
	}
	if !strings.Contains(app, "Strike <span>beats Feint</span>") {
		problems = append(problems, "overview does not say Strike beats Feint")
	}
	if !strings.Contains(app, "not the live pass-line ledger") {
		problems = append(problems, "overview does not say the archive is separate from the live ledger")
	}
	if strings.Contains(store, "INTO bots") || strings.Contains(store, "marble_ledger") || strings.Contains(store, "SET marbles") {
		problems = append(problems, "spectator SQL writes the settlement ledger")
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
