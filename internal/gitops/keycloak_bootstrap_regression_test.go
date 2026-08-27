package gitops

import (
	"path/filepath"
	"testing"
)

func TestKeycloakPostgresRendersKeycloakNamespace(t *testing.T) {
	dst := t.TempDir()
	cfg := newDefault("keycloak-postgres-ns")
	cfg.OpenCenter.GitOps.Repository.LocalDir = dst

	if err := RenderClusterApps(cfg); err != nil {
		t.Fatalf("RenderClusterApps() error = %v", err)
	}

	postgresDir := filepath.Join(dst, "applications", "overlays", cfg.ClusterName(), "services", "keycloak", "00-postgres")

	kustomizationDocs, err := decodeYAMLDocuments([]byte(mustReadFile(t, filepath.Join(postgresDir, "kustomization.yaml"))))
	if err != nil {
		t.Fatalf("parse kustomization.yaml: %v", err)
	}
	resources, ok := nestedValue(kustomizationDocs[0], "resources").([]any)
	if !ok || len(resources) < 2 {
		t.Fatalf("kustomization resources = %#v, want namespace then postgres-cluster", resources)
	}
	if resources[0] != "./namespace.yaml" {
		t.Fatalf("kustomization resources[0] = %#v, want ./namespace.yaml", resources[0])
	}

	nsDocs, err := decodeYAMLDocuments([]byte(mustReadFile(t, filepath.Join(postgresDir, "namespace.yaml"))))
	if err != nil {
		t.Fatalf("parse namespace.yaml: %v", err)
	}
	if len(nsDocs) != 1 {
		t.Fatalf("namespace.yaml documents = %d, want 1", len(nsDocs))
	}
	ns := nsDocs[0]
	if got := ns["kind"]; got != "Namespace" {
		t.Errorf("namespace.yaml kind = %#v, want Namespace", got)
	}
	if got := nestedString(ns, "metadata", "name"); got != "keycloak" {
		t.Errorf("namespace.yaml metadata.name = %q, want keycloak", got)
	}
}

func TestKeycloakOperatorScopedToKeycloakNamespace(t *testing.T) {
	dst := t.TempDir()
	cfg := newDefault("keycloak-operator-scope")
	cfg.OpenCenter.GitOps.Repository.LocalDir = dst

	if err := RenderClusterApps(cfg); err != nil {
		t.Fatalf("RenderClusterApps() error = %v", err)
	}

	operatorDir := filepath.Join(dst, "applications", "overlays", cfg.ClusterName(), "services", "keycloak", "10-operator")

	kustomizationDocs, err := decodeYAMLDocuments([]byte(mustReadFile(t, filepath.Join(operatorDir, "kustomization.yaml"))))
	if err != nil {
		t.Fatalf("parse kustomization.yaml: %v", err)
	}
	resources, ok := nestedValue(kustomizationDocs[0], "resources").([]any)
	if !ok {
		t.Fatalf("kustomization resources = %#v", resources)
	}
	found := map[string]bool{}
	for _, r := range resources {
		found[r.(string)] = true
	}
	if !found["./operator-group.yaml"] {
		t.Errorf("kustomization resources missing ./operator-group.yaml: %#v", resources)
	}
	if !found["./patch-subscription.yaml"] {
		t.Errorf("kustomization resources missing ./patch-subscription.yaml: %#v", resources)
	}

	ogDocs, err := decodeYAMLDocuments([]byte(mustReadFile(t, filepath.Join(operatorDir, "operator-group.yaml"))))
	if err != nil {
		t.Fatalf("parse operator-group.yaml: %v", err)
	}
	if len(ogDocs) != 1 {
		t.Fatalf("operator-group.yaml documents = %d, want 1", len(ogDocs))
	}
	og := ogDocs[0]
	if got := og["kind"]; got != "OperatorGroup" {
		t.Errorf("operator-group.yaml kind = %#v, want OperatorGroup", got)
	}
	if got := nestedString(og, "metadata", "namespace"); got != "keycloak" {
		t.Errorf("OperatorGroup metadata.namespace = %q, want keycloak", got)
	}
	targets, ok := nestedValue(og, "spec", "targetNamespaces").([]any)
	if !ok || len(targets) != 1 || targets[0] != "keycloak" {
		t.Fatalf("OperatorGroup spec.targetNamespaces = %#v, want [keycloak]", targets)
	}

	subDocs, err := decodeYAMLDocuments([]byte(mustReadFile(t, filepath.Join(operatorDir, "patch-subscription.yaml"))))
	if err != nil {
		t.Fatalf("parse patch-subscription.yaml: %v", err)
	}
	if got := nestedString(subDocs[0], "metadata", "namespace"); got != "keycloak" {
		t.Errorf("Subscription metadata.namespace = %q, want keycloak", got)
	}

	fluxPath := filepath.Join(dst, "applications", "overlays", cfg.ClusterName(), "services", "fluxcd", "keycloak.yaml")
	fluxDocs, err := decodeYAMLDocuments([]byte(mustReadFile(t, fluxPath)))
	if err != nil {
		t.Fatalf("parse %s: %v", fluxPath, err)
	}
	operatorStage := findFluxKustomization(t, fluxDocs, "keycloak-operator")
	if got := nestedString(operatorStage, "spec", "targetNamespace"); got != "keycloak" {
		t.Errorf("keycloak-operator Kustomization targetNamespace = %q, want keycloak", got)
	}
	healthChecks, ok := nestedValue(operatorStage, "spec", "healthChecks").([]any)
	if !ok || len(healthChecks) != 1 {
		t.Fatalf("keycloak-operator healthChecks = %#v, want one Deployment health check", healthChecks)
	}
	hc := healthChecks[0].(map[string]any)
	if got := hc["namespace"]; got != "keycloak" {
		t.Errorf("keycloak-operator healthCheck namespace = %q, want keycloak", got)
	}
}
