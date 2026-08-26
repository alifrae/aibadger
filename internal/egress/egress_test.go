package egress

import (
	"strings"
	"testing"

	"github.com/PVRLabs/aibadger/internal/projectpolicy"
)

func TestInspectPolicyPaths(t *testing.T) {
	policy := projectpolicy.Policy{Security: projectpolicy.SecurityPolicy{
		Deny: []string{"recordings/**"},
		Warn: []string{"calibration/**"},
	}}
	if decision := Inspect("recordings/run.dat", "safe", policy); !decision.Blocked {
		t.Fatal("expected denied path to be blocked")
	}
	decision := Inspect("calibration/config.yaml", "safe", policy)
	if decision.Blocked || decision.Warning == "" {
		t.Fatalf("expected warning-only decision: %+v", decision)
	}
}

func TestInspectHighConfidenceSecret(t *testing.T) {
	policy := projectpolicy.Policy{Security: projectpolicy.SecurityPolicy{BlockSecrets: true}}
	secret := "token = \"sk-" + strings.Repeat("A", 28) + "\""
	decision := Inspect("src/config.py", secret, policy)
	if !decision.Blocked {
		t.Fatalf("expected secret to be blocked: %+v", decision)
	}
}

func TestInspectPlaceholderIsNotBlocked(t *testing.T) {
	policy := projectpolicy.Policy{Security: projectpolicy.SecurityPolicy{BlockSecrets: true}}
	decision := Inspect("src/config.py", `api_key = "your_api_key_placeholder_value"`, policy)
	if decision.Blocked {
		t.Fatalf("placeholder should not be blocked: %+v", decision)
	}
}
