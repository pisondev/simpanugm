package config

import (
	"strings"
	"testing"
)

func base() Config {
	return Config{
		AppEnv: "development", PublicBaseURL: "http://localhost:8000", StorageDriver: "fs", SMTPTLS: "none",
		SessionCookieSecure: true, WorkerQueues: "default=10,ml=2", TrustedProxies: []string{"10.0.0.0/8"},
		HandoverPepper:      "unit-test-pepper-0123456789",
		LoginIPPer15Minutes: 20, LoginPairPer15Minutes: 5,
	}
}

func TestLoginIPBudgetFromEnvironment(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("PUBLIC_BASE_URL", "http://localhost:8000")
	t.Setenv("DATABASE_URL", "postgres://localhost/simpan")
	t.Setenv("AUTH_LOGIN_PAIR_PER_15_MINUTES", "5")
	for _, tc := range []struct {
		name, value string
		want        int
	}{
		{name: "default", value: "", want: 20},
		{name: "configured", value: "500", want: 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("AUTH_LOGIN_IP_PER_15_MINUTES", tc.value)
			c, err := Load()
			if err != nil || c.LoginIPPer15Minutes != tc.want {
				t.Fatalf("login IP budget: got %d, want %d, error %v", c.LoginIPPer15Minutes, tc.want, err)
			}
		})
	}
	t.Setenv("AUTH_LOGIN_IP_PER_15_MINUTES", "0")
	if _, err := Load(); err == nil {
		t.Fatal("zero login budget accepted")
	}
	t.Setenv("AUTH_LOGIN_IP_PER_15_MINUTES", "20")
	t.Setenv("AUTH_LOGIN_PAIR_PER_15_MINUTES", "500")
	if c, err := Load(); err != nil || c.LoginPairPer15Minutes != 500 {
		t.Fatalf("configured pair budget: got %d error %v", c.LoginPairPer15Minutes, err)
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
