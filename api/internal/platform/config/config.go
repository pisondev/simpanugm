package config

import (
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"strings"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	AppEnv        string `env:"APP_ENV,required"`
	HTTPAddr      string `env:"HTTP_ADDR" envDefault:":8090"`
	MetricsAddr   string `env:"METRICS_ADDR" envDefault:":9091"`
	PublicBaseURL string `env:"PUBLIC_BASE_URL,required"`

	DatabaseURL         string `env:"DATABASE_URL,required"`
	DatabaseMigratorURL string `env:"DATABASE_MIGRATOR_URL"`
	DatabaseMaxConns    int32  `env:"DATABASE_MAX_CONNS" envDefault:"20"`

	StorageDriver string `env:"STORAGE_DRIVER" envDefault:"fs"`
	StorageFSRoot string `env:"STORAGE_FS_ROOT" envDefault:"./var/media"`

	// BR-HND-01, the pepper of the handover codes, at least 16 characters
	HandoverPepper string `env:"HANDOVER_PEPPER"`

	MLBaseURL       string `env:"ML_BASE_URL"`
	MLInternalToken string `env:"ML_INTERNAL_TOKEN"`

	SMTPHost     string `env:"SMTP_HOST" envDefault:"localhost"`
	SMTPPort     int    `env:"SMTP_PORT" envDefault:"1025"`
	SMTPUsername string `env:"SMTP_USERNAME"`
	SMTPPassword string `env:"SMTP_PASSWORD"`
	SMTPTLS      string `env:"SMTP_TLS" envDefault:"starttls"`
	MailFrom     string `env:"MAIL_FROM" envDefault:"SimpanUGM <no-reply@simpanugm.my.id>"`

	SessionCookieSecure bool     `env:"SESSION_COOKIE_SECURE" envDefault:"true"`
	TrustedProxies      []string `env:"TRUSTED_PROXIES" envSeparator:"," envDefault:"127.0.0.1/32,::1/128,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16"`
	WorkerQueues        string   `env:"WORKER_QUEUES" envDefault:"default=10,ml=2,email=2,maintenance=1"`
	LogLevel            string   `env:"LOG_LEVEL" envDefault:"info"`
	FeatureFakeClock    bool     `env:"FEATURE_FAKE_CLOCK" envDefault:"false"`
}

var validEnvs = map[string]bool{"development": true, "test": true, "e2e": true, "staging": true, "production": true}

func Load() (Config, error) {
	var c Config
	if err := env.Parse(&c); err != nil {
		return c, err
	}
	return c, c.validate()
}

func (c Config) validate() error {
	var errs []error
	if !validEnvs[c.AppEnv] {
		errs = append(errs, fmt.Errorf("APP_ENV %q is not one of development, test, e2e, staging, production", c.AppEnv))
	}
	if u, err := url.Parse(c.PublicBaseURL); err != nil || u.Scheme == "" || u.Host == "" {
		errs = append(errs, fmt.Errorf("PUBLIC_BASE_URL %q must be an absolute URL", c.PublicBaseURL))
	}
	if c.StorageDriver != "fs" {
		errs = append(errs, fmt.Errorf("STORAGE_DRIVER %q is not supported, use fs", c.StorageDriver))
	}
	switch c.SMTPTLS {
	case "starttls", "tls", "none":
	default:
		errs = append(errs, fmt.Errorf("SMTP_TLS %q must be starttls, tls or none", c.SMTPTLS))
	}
	if len(c.HandoverPepper) < 16 && c.AppEnv != "test" {
		errs = append(errs, errors.New("HANDOVER_PEPPER must be at least 16 characters"))
	}
	if c.FeatureFakeClock && c.AppEnv != "e2e" {
		errs = append(errs, errors.New("FEATURE_FAKE_CLOCK is allowed only when APP_ENV=e2e"))
	}
	if !c.SessionCookieSecure && c.AppEnv != "development" && c.AppEnv != "test" && c.AppEnv != "e2e" {
		errs = append(errs, errors.New("SESSION_COOKIE_SECURE=false is allowed only in development, test and e2e"))
	}
	if _, err := c.TrustedPrefixes(); err != nil {
		errs = append(errs, err)
	}
	if _, err := c.Queues(); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func (c Config) IsProduction() bool { return c.AppEnv == "production" }

func (c Config) TrustedPrefixes() ([]netip.Prefix, error) {
	out := make([]netip.Prefix, 0, len(c.TrustedProxies))
	for _, s := range c.TrustedProxies {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		p, err := netip.ParsePrefix(s)
		if err != nil {
			return nil, fmt.Errorf("TRUSTED_PROXIES: %w", err)
		}
		out = append(out, p)
	}
	return out, nil
}

func (c Config) Queues() (map[string]int, error) {
	out := map[string]int{}
	for _, part := range strings.Split(c.WorkerQueues, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		name, n, ok := strings.Cut(part, "=")
		var workers int
		if _, err := fmt.Sscanf(n, "%d", &workers); !ok || err != nil || workers < 1 {
			return nil, fmt.Errorf("WORKER_QUEUES entry %q must look like name=workers", part)
		}
		out[strings.TrimSpace(name)] = workers
	}
	return out, nil
}
