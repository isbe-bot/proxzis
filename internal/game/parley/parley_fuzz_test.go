package parley

import (
	"testing"
)

func FuzzParleyRoundResolution(f *testing.F) {
	// Seed corpus with canonical stance pairs and modifiers
	stances := []string{"rock", "paper", "scissors", "ROCK", "PAPER", "SCISSORS", "", "invalid", "foo"}
	modifiers := []string{"", "MirrorTiePenalty", "ScissorsSurge", "PaperRebound", "invalid_mod"}

	for _, s1 := range stances {
		for _, s2 := range stances {
			for _, m := range modifiers {
				f.Add(s1, s2, m)
			}
		}
	}

	f.Fuzz(func(t *testing.T, rawS1 string, rawS2 string, rawMod string) {
		s1, err1 := ParseStance(rawS1)
		s2, err2 := ParseStance(rawS2)
		mod, errMod := ParseModifierCard(rawMod)

		res, err := ResolveRound(s1, s2, mod)

		if err1 != nil || err2 != nil {
			if err == nil {
				t.Fatalf("expected error for invalid stance inputs: s1=%q, s2=%q", rawS1, rawS2)
			}
			return
		}

		// Valid stances must resolve cleanly without panic
		if err != nil {
			t.Fatalf("unexpected error resolving valid stances %v vs %v: %v", s1, s2, err)
		}

		// Invariant checks:
		if s1 == s2 {
			if !res.IsTie {
				t.Errorf("equal stances must result in tie")
			}
			if res.P1Points != 0 || res.P2Points != 0 {
				t.Errorf("tie must award 0 points, got %d, %d", res.P1Points, res.P2Points)
			}
			if mod == ModifierMirrorTiePenalty {
				if res.MirrorPenP1 != 1 || res.MirrorPenP2 != 1 {
					t.Errorf("MirrorTiePenalty must assess 1 penalty each on tie")
				}
			}
		} else {
			if res.IsTie {
				t.Errorf("different stances must not result in tie")
			}
			if res.Winner != Player1 && res.Winner != Player2 {
				t.Errorf("non-tie must have a winner")
			}
			if res.P1Points < 0 || res.P2Points < 0 {
				t.Errorf("points cannot be negative")
			}
		}

		// Mod sanity check
		if errMod == nil && mod == ModifierScissorsSurge {
			if res.Winner == Player1 && s1 == StanceScissors && res.P1Points != 2 {
				t.Errorf("ScissorsSurge must award 2 pts to P1 scissors win")
			}
			if res.Winner == Player2 && s2 == StanceScissors && res.P2Points != 2 {
				t.Errorf("ScissorsSurge must award 2 pts to P2 scissors win")
			}
		}
	})
}
