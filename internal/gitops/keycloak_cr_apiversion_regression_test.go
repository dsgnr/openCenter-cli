package gitops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestKeycloakCRUsesV2Beta1APIVersion(t *testing.T) {
	dst := t.TempDir()
	cfg := newDefault("keycloak-cr-apiversion")
	cfg.OpenCenter.GitOps.Repository.LocalDir = dst

	if err := RenderClusterApps(cfg); err != nil {
		t.Fatalf("RenderClusterApps() error = %v", err)
	}

	keycloakDir := filepath.Join(dst, "applications", "overlays", cfg.ClusterName(), "services", "keycloak", "20-keycloak")

	cases := []struct {
		file string
		want string
	}{
		{"keycloak-cr-patch.yaml", "apiVersion: k8s.keycloak.org/v2beta1"},
		{"opencenter-realm.yaml", "apiVersion: k8s.keycloak.org/v2beta1"},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			content := mustReadFile(t, filepath.Join(keycloakDir, tc.file))
			if !strings.Contains(content, tc.want) {
				t.Fatalf("%s missing %q", tc.file, tc.want)
			}
			if strings.Contains(content, "k8s.keycloak.org/v2alpha1") {
				t.Fatalf("%s still references deprecated k8s.keycloak.org/v2alpha1", tc.file)
			}
		})
	}

	hpaPath := filepath.Join(keycloakDir, "keycloak-hpa.yaml")
	if _, err := os.Stat(hpaPath); err == nil {
		hpaContent := mustReadFile(t, hpaPath)
		if strings.Contains(hpaContent, "k8s.keycloak.org/v2alpha1") {
			t.Fatalf("keycloak-hpa.yaml scaleTargetRef still references deprecated k8s.keycloak.org/v2alpha1")
		}
	}
}
