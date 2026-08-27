package gitops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPromoteOverlayClaimsExistingFileWhenPlanned(t *testing.T) {
	root := t.TempDir()
	workspace := t.TempDir()

	planned := filepath.Join(workspace, "services", "postgres-operator", "kustomization.yaml")
	writeTestFile(t, planned, "generated: v2")

	existing := filepath.Join(root, "services", "postgres-operator", "kustomization.yaml")
	writeTestFile(t, existing, "generated: v1")

	result, err := promoteOverlay(workspace, root, "cluster", PromoteOptions{})
	if err != nil {
		t.Fatalf("promoteOverlay() unexpectedly failed: %v", err)
	}

	got, err := os.ReadFile(existing)
	if err != nil {
		t.Fatalf("read existing file after promote: %v", err)
	}
	if string(got) != "generated: v2" {
		t.Fatalf("existing file was not overwritten: got %q", string(got))
	}

	manifest := readTestManifest(t, root)
	if _, tracked := manifest.Files["services/postgres-operator/kustomization.yaml"]; !tracked {
		t.Fatal("planned file was overwritten but not recorded in the generator manifest")
	}

	for _, warning := range result.Warnings {
		if strings.Contains(warning, "user-authored") || strings.Contains(warning, "Force") {
			t.Fatalf("unexpected warning about a planned file: %q", warning)
		}
	}
}
