package config

import (
	"strings"
	"testing"
)

func base() Config {
	return Config{
		AppEnv: "development", PublicBaseURL: "http://localhost:8000", StorageDriver: "fs", SMTPTLS: "none",
		SessionCookieSecure: true, WorkerQueues: "default=10,ml=2", TrustedProxies: []string{"10.0.0.0/8"},
	}
}

func TestQueues(t *testing.T) {
	q, err := base().Queues()
	if err != nil || q["default"] != 10 || q["ml"] != 2 {
		t.Fatalf("queues %v %v", q, err)
	}
	c := base()
	c.WorkerQueues = "default=zero"
	if _, err := c.Queues(); err == nil {
		t.Fatal("bad queue spec accepted")
	}
}

func TestValidate(t *testing.T) {
	if err := base().validate(); err != nil {
		t.Fatal(err)
	}
	c := base()
	c.AppEnv, c.SessionCookieSecure, c.FeatureFakeClock = "production", false, true
	err := c.validate()
	if err == nil || !strings.Contains(err.Error(), "FEATURE_FAKE_CLOCK") || !strings.Contains(err.Error(), "SESSION_COOKIE_SECURE") {
		t.Fatalf("production guards: %v", err)
	}
}
