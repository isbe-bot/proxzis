package config

import "testing"

func TestDatabaseCredentialsAreNeverDefaulted(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	if Load().DatabaseURL != "" {
		t.Fatal("implicit database credential")
	}
	t.Setenv("DATABASE_URL", "postgres://explicit-config/test")
	if Load().DatabaseURL != "postgres://explicit-config/test" {
		t.Fatal("explicit URL lost")
	}
}

func TestPortBounds(t *testing.T) {
	for _, value := range []string{"-1", "0", "65536", "invalid"} {
		t.Setenv("PORT", value)
		if Load().Port != 8080 {
			t.Errorf("invalid port %q accepted", value)
		}
	}
	t.Setenv("PORT", "9000")
	if Load().Port != 9000 {
		t.Fatal("valid port rejected")
	}
}
