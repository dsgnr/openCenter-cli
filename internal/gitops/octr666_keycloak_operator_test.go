package gitops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOCTR666KeycloakOperatorRendersForNonOrd1(t *testing.T) {
	dst := t.TempDir()
	cfg := newDefault("octr666-keycloak")
	cfg.OpenCenter.Meta.Region = "dfw"
	cfg.OpenCenter.GitOps.Repository.LocalDir = dst

	if err := RenderClusterApps(cfg); err != nil {
		t.Fatalf("RenderClusterApps() error = %v", err)
	}

	operatorDir := filepath.Join(dst, "applications", "overlays", cfg.ClusterName(), "services", "keycloak", "10-operator")
	kustomizationPath := filepath.Join(operatorDir, "kustomization.yaml")
	kustomizationDocs, err := decodeYAMLDocuments([]byte(mustReadFile(t, kustomizationPath)))
	if err != nil {
		t.Fatalf("parse %s: %v", kustomizationPath, err)
	}
	if len(kustomizationDocs) != 1 {
		t.Fatalf("expected one Kustomization document in %s, got %d", kustomizationPath, len(kustomizationDocs))
	}
	resources, ok := nestedValue(kustomizationDocs[0], "resources").([]any)
	wantResources := []any{"./operator-group.yaml", "./patch-subscription.yaml"}
	if !ok || len(resources) != len(wantResources) {
		t.Fatalf("%s resources = %#v, want %#v", kustomizationPath, resources, wantResources)
	}
	for i, r := range wantResources {
		if resources[i] != r {
			t.Fatalf("%s resources[%d] = %#v, want %#v", kustomizationPath, i, resources[i], r)
		}
	}

	subscriptionPath := filepath.Join(operatorDir, "patch-subscription.yaml")
	subscriptionContent := mustReadFile(t, subscriptionPath)
	subscriptionDocs, err := decodeYAMLDocuments([]byte(subscriptionContent))
	if err != nil {
		t.Fatalf("parse %s: %v", subscriptionPath, err)
	}
	if len(subscriptionDocs) != 1 {
		t.Fatalf("expected one Subscription document in %s, got %d", subscriptionPath, len(subscriptionDocs))
	}
	subscription := subscriptionDocs[0]
	if got := subscription["apiVersion"]; got != "operators.coreos.com/v1alpha1" {
		t.Errorf("Subscription apiVersion = %#v, want operators.coreos.com/v1alpha1", got)
	}
	if got := subscription["kind"]; got != "Subscription" {
		t.Errorf("operator resource kind = %#v, want Subscription", got)
	}
	if got := nestedString(subscription, "metadata", "name"); got != "keycloak-operator" {
		t.Errorf("Subscription metadata.name = %q, want keycloak-operator", got)
	}
	if got := nestedString(subscription, "metadata", "namespace"); got != "keycloak" {
		t.Errorf("Subscription metadata.namespace = %q, want keycloak", got)
	}
	for field, want := range map[string]string{
		"name":                "keycloak-operator",
		"channel":             "fast",
		"source":              "operatorhubio-catalog",
		"sourceNamespace":     "olm",
		"startingCSV":         "keycloak-operator.v26.4.2",
		"installPlanApproval": "Automatic",
	} {
		if got := nestedString(subscription, "spec", field); got != want {
			t.Errorf("Subscription spec.%s = %q, want %q", field, got, want)
		}
	}
	assertOrderedSubstrings(t, subscriptionContent,
		"name: keycloak-operator",
		"channel: fast",
		"source: operatorhubio-catalog",
		"sourceNamespace: olm",
		"startingCSV: keycloak-operator.v26.4.2",
		"installPlanApproval: Automatic",
	)

	entries, err := os.ReadDir(operatorDir)
	if err != nil {
		t.Fatalf("read %s: %v", operatorDir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		docs, err := decodeYAMLDocuments([]byte(mustReadFile(t, filepath.Join(operatorDir, entry.Name()))))
		if err != nil {
			t.Fatalf("parse generated operator resource %s: %v", entry.Name(), err)
		}
		for _, doc := range docs {
			if doc["kind"] == "CatalogSource" {
				t.Fatalf("generated Keycloak operator resources must not include a CatalogSource: %s", entry.Name())
			}
		}
	}

	fluxPath := filepath.Join(dst, "applications", "overlays", cfg.ClusterName(), "services", "fluxcd", "keycloak.yaml")
	fluxDocs, err := decodeYAMLDocuments([]byte(mustReadFile(t, fluxPath)))
	if err != nil {
		t.Fatalf("parse %s: %v", fluxPath, err)
	}
	operatorStage := findFluxKustomization(t, fluxDocs, "keycloak-operator")
	assertFluxDependenciesInOrder(t, operatorStage, "keycloak-operator", "sources", "olm-base", "keycloak-postgres")
	if got := nestedString(operatorStage, "spec", "targetNamespace"); got != "keycloak" {
		t.Errorf("keycloak-operator targetNamespace = %q, want keycloak", got)
	}
	healthChecks, ok := nestedValue(operatorStage, "spec", "healthChecks").([]any)
	if !ok || len(healthChecks) != 1 {
		t.Fatalf("keycloak-operator healthChecks = %#v, want one Deployment health check", healthChecks)
	}
	healthCheck, ok := healthChecks[0].(map[string]any)
	if !ok {
		t.Fatalf("keycloak-operator health check has unexpected shape: %#v", healthChecks[0])
	}
	for field, want := range map[string]string{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"name":       "keycloak-operator",
		"namespace":  "keycloak",
	} {
		if got := healthCheck[field]; got != want {
			t.Errorf("keycloak-operator health check %s = %#v, want %q", field, got, want)
		}
	}

	keycloakStage := findFluxKustomization(t, fluxDocs, "keycloak-cr")
	if !hasFluxDependency(t, keycloakStage, "keycloak-operator") {
		t.Fatalf("keycloak-cr must depend on keycloak-operator")
	}
}

func assertOrderedSubstrings(t *testing.T, content string, want ...string) {
	t.Helper()
	previous := -1
	for _, substring := range want {
		index := strings.Index(content, substring)
		if index < 0 {
			t.Errorf("rendered content is missing %q", substring)
			continue
		}
		if index <= previous {
			t.Errorf("rendered content places %q out of order", substring)
		}
		previous = index
	}
}

func assertFluxDependenciesInOrder(t *testing.T, doc map[string]any, stage string, wanted ...string) {
	t.Helper()
	dependencies, ok := nestedValue(doc, "spec", "dependsOn").([]any)
	if !ok {
		t.Fatalf("%s has no spec.dependsOn list: %#v", stage, doc)
	}
	if len(dependencies) != len(wanted) {
		t.Fatalf("%s dependencies = %#v, want names in order %v", stage, dependencies, wanted)
	}
	for index, raw := range dependencies {
		dependency, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("%s has malformed dependency: %#v", stage, raw)
		}
		if got := dependency["name"]; got != wanted[index] {
			t.Errorf("%s dependency %d = %#v, want %q", stage, index, got, wanted[index])
		}
	}
}
