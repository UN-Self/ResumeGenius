package database

import "testing"

func TestBuildDSN(t *testing.T) {
	dsn := buildDSN("localhost", "5432", "testuser", "testpass", "testdb")
	expected := "host=localhost port=5432 user=testuser password=testpass dbname=testdb sslmode=disable"
	if dsn != expected {
		t.Errorf("got %s, want %s", dsn, expected)
	}
}

func TestBuildDSNWithEnvDefaults(t *testing.T) {
	dsn := buildDSN("myhost", "5433", "admin", "secret", "mydb")
	if dsn != "host=myhost port=5433 user=admin password=secret dbname=mydb sslmode=disable" {
		t.Errorf("got %s", dsn)
	}
}

func TestEnvOrDefault(t *testing.T) {
	t.Setenv("TEST_KEY", "from_env")
	if v := envOrDefault("TEST_KEY", "default"); v != "from_env" {
		t.Errorf("expected from_env, got %s", v)
	}
	if v := envOrDefault("MISSING_KEY", "fallback"); v != "fallback" {
		t.Errorf("expected fallback, got %s", v)
	}
}
