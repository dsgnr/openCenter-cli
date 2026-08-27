package gitops

import (
	"path/filepath"
	"testing"
)

func TestKeycloakHostnameSpreadUsesScheduleAnyway(t *testing.T) {
	dst := t.TempDir()
	cfg := newDefault("keycloak-topology")
	cfg.OpenCenter.GitOps.Repository.LocalDir = dst

	if err := RenderClusterApps(cfg); err != nil {
		t.Fatalf("RenderClusterApps() error = %v", err)
	}

	crPath := filepath.Join(dst, "applications", "overlays", cfg.ClusterName(), "services", "keycloak", "20-keycloak", "keycloak-cr-patch.yaml")
	docs, err := decodeYAMLDocuments([]byte(mustReadFile(t, crPath)))
	if err != nil {
		t.Fatalf("parse %s: %v", crPath, err)
	}

	var kc map[string]any
	for _, d := range docs {
		if d["kind"] == "Keycloak" {
			kc = d
			break
		}
	}
	if kc == nil {
		t.Fatalf("Keycloak CR not found in %s", crPath)
	}

	constraints, ok := nestedValue(kc, "spec", "unsupported", "podTemplate", "spec", "topologySpreadConstraints").([]any)
	if !ok {
		t.Fatalf("topologySpreadConstraints missing or wrong shape: %#v", constraints)
	}

	found := false
	for _, raw := range constraints {
		c, _ := raw.(map[string]any)
		if c["topologyKey"] != "kubernetes.io/hostname" {
			continue
		}
		found = true
		if got := c["whenUnsatisfiable"]; got != "ScheduleAnyway" {
			t.Fatalf("hostname topology spread whenUnsatisfiable = %q, want ScheduleAnyway", got)
		}
	}
	if !found {
		t.Fatalf("hostname topology spread constraint not found in %#v", constraints)
	}
}
