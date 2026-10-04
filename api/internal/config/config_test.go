package config

import "testing"

// base is the minimum environment that boots in production.
func base() map[string]string {
	return map[string]string{
		"DATABASE_URL":              "postgres://user:password@db:5432/shule360",
		"SUPABASE_URL":              "https://project.supabase.co",
		"SUPABASE_SERVICE_ROLE_KEY": "service-role-key",
		"JWT_SECRET":                "a-real-secret",
		"APP_ENV":                   "production",
	}
}

func load(env map[string]string) (*Config, error) {
	return LoadFrom(func(k string) string { return env[k] })
}

// TestMpesaPlaceholderDoesNotBlockBoot is the regression this change exists
// for: a service whose Daraja credentials are still the .env.example
// placeholders used to refuse to boot, taking SMS down with it.
func TestMpesaPlaceholderDoesNotBlockBoot(t *testing.T) {
	env := base()
	// A real consumer key next to an unfilled passkey, which is the state the
	// service was actually in.
	env["MPESA_CONSUMER_KEY"] = "Zq3Lw8Rt5Yb1Nc6Vd9Mf2Hg7Jk4Pp0Sa8Xe3Ui6OoTy1Bn"
	env["MPESA_PASSKEY"] = "your_mpesa_passkey_here"
	// Deliberately no MPESA_ALLOWED_IPS.

	cfg, err := load(env)
	if err != nil {
		t.Fatalf("boot must not be blocked by an unfilled M-Pesa placeholder: %v", err)
	}
	if cfg.IsMpesaEnabled() {
		t.Error("M-Pesa must not count as enabled while the passkey is a placeholder")
	}
}

// TestMpesaExplicitlyDisabledBoots covers the supported way to hold payments
// back without deleting the credentials.
func TestMpesaExplicitlyDisabledBoots(t *testing.T) {
	for _, v := range []string{"false", "FALSE", "0", "no", "off", " false "} {
		env := base()
		env["MPESA_ENABLED"] = v
		env["MPESA_CONSUMER_KEY"] = "real-consumer-key"
		env["MPESA_PASSKEY"] = "real-passkey"

		cfg, err := load(env)
		if err != nil {
			t.Errorf("MPESA_ENABLED=%q: boot must succeed, got %v", v, err)
			continue
		}
		if cfg.IsMpesaEnabled() {
			t.Errorf("MPESA_ENABLED=%q: must be treated as disabled", v)
		}
	}
}

// TestMpesaEnabledRequiresAllowlist is the guard that must not be weakened: a
// live Daraja integration with no IP allowlist exposes an unauthenticated
// payment callback, so this has to keep failing in production.
func TestMpesaEnabledRequiresAllowlist(t *testing.T) {
	env := base()
	env["MPESA_CONSUMER_KEY"] = "real-consumer-key"
	env["MPESA_PASSKEY"] = "real-passkey"

	if _, err := load(env); err == nil {
		t.Fatal("production boot must fail when M-Pesa is live without MPESA_ALLOWED_IPS")
	}

	env["MPESA_ALLOWED_IPS"] = "196.201.214.0/24"
	cfg, err := load(env)
	if err != nil {
		t.Fatalf("with an allowlist configured, boot must succeed: %v", err)
	}
	if !cfg.IsMpesaEnabled() {
		t.Error("real credentials must be inferred as enabled")
	}
}

// TestMpesaExplicitlyEnabledStillNeedsAllowlist makes sure turning the switch on
// by hand cannot bypass the guard.
func TestMpesaExplicitlyEnabledStillNeedsAllowlist(t *testing.T) {
	env := base()
	env["MPESA_ENABLED"] = "true"
	env["MPESA_ALLOWED_IPS"] = "196.201.214.0/24"

	if _, err := load(env); err != nil {
		t.Fatalf("MPESA_ENABLED=true with an allowlist must boot: %v", err)
	}

	delete(env, "MPESA_ALLOWED_IPS")
	if _, err := load(env); err == nil {
		t.Error("MPESA_ENABLED=true without an allowlist must still fail in production")
	}
}

// TestLooksPlaceholderKeepsAmbiguousMarkers guards the conservative direction:
// misreading a real base64 credential as a placeholder would silently disable
// the allowlist guard.
func TestLooksPlaceholderKeepsAmbiguousMarkers(t *testing.T) {
	placeholders := []string{
		"your_mpesa_passkey_here", "your-consumer-key", "CHANGEME",
		"change-me", "placeholder", "<your_key>", "TODO",
	}
	for _, v := range placeholders {
		if !looksPlaceholder(v) {
			t.Errorf("looksPlaceholder(%q) = false, want true", v)
		}
	}

	// Made-up strings shaped like a Daraja consumer key (never a real one, not
	// even part of one): no marker, and it contains "xxx" in a
	// position where base64url could plausibly produce it.
	reals := []string{
		"Zq3Lw8Rt5Yb1Nc6Vd9Mf2Hg7Jk4Pp0Sa8Xe3Ui6OoTy1Bn",
		"Kd7Fs2Gh9Jl4Qw1Er6Ty3Ui8Op5As0Df2Gh7Jk4Lz9Xc",
		"abcxxxdef",
	}
	for _, v := range reals {
		if looksPlaceholder(v) {
			t.Errorf("looksPlaceholder(%q) = true, want false", v)
		}
	}
}

// TestMissingJWTSecretStillBlocksBoot keeps the check that caused the outage
// in the first place.
func TestMissingJWTSecretStillBlocksBoot(t *testing.T) {
	env := base()
	delete(env, "JWT_SECRET")
	if _, err := load(env); err == nil {
		t.Fatal("production boot must fail without JWT_SECRET")
	}
}
