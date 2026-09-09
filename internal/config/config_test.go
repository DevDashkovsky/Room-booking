package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	cfg := Load()
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want 8080", cfg.Port)
	}
	if cfg.JWTSecret == "" {
		t.Error("JWTSecret is empty")
	}
	if cfg.DatabaseURL == "" {
		t.Error("DatabaseURL is empty")
	}
}

func TestLoad_FromEnv(t *testing.T) {
	os.Setenv("PORT", "9090")
	defer os.Unsetenv("PORT")

	cfg := Load()
	if cfg.Port != "9090" {
		t.Errorf("Port = %q, want 9090", cfg.Port)
	}
}

func TestLoad_DBMaxConns(t *testing.T) {
	for _, tt := range []struct {
		value string
		want  int32
	}{{"", 20}, {"30", 30}, {"0", 20}, {"-1", 20}, {"invalid", 20}, {"2147483648", 20}, {"4294967297", 20}} {
		t.Run(tt.value, func(t *testing.T) {
			t.Setenv("DB_MAX_CONNS", tt.value)
			if got := Load().DBMaxConns; got != tt.want {
				t.Fatalf("got %d want %d", got, tt.want)
			}
		})
	}
}
