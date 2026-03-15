package internal_test

import (
	"strings"
	"testing"

	"github.com/compositor/kompoze/internal/converter"
	"github.com/compositor/kompoze/internal/parser"
	"github.com/compositor/kompoze/internal/output"
	"github.com/compositor/kompoze/internal/validator"
)

func defaultOpts() converter.ConvertOptions {
	return converter.ConvertOptions{
		OutputDir:        "./test-output",
		Namespace:        "test",
		AppName:          "test-app",
		AddProbes:        true,
		AddResources:     true,
		AddSecurity:      true,
		AddNetworkPolicy: true,
	}
}

func TestIntegrationSimpleCompose(t *testing.T) {
	compose, err := parser.ParseComposeFile("../testdata/simple-compose.yml")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	result, err := converter.Convert(compose, defaultOpts())
	if err != nil {
		t.Fatalf("convert error: %v", err)
	}

	// Should have 2 services: cache (redis) and web (nginx)
	if len(result.Deployments) != 2 {
		t.Errorf("expected 2 deployments, got %d", len(result.Deployments))
	}

	// web should have a Service (has ports), cache should have a Service (has ports)
	if len(result.Services) != 2 {
		t.Errorf("expected 2 services, got %d", len(result.Services))
	}

	// Validate
	vErrors := validator.ValidateManifests(result)
	if validator.HasErrors(vErrors) {
		t.Errorf("validation errors: %v", vErrors)
	}

	// Render should succeed
	rendered, err := output.RenderManifests(result)
	if err != nil {
		t.Fatalf("render error: %v", err)
	}
	if !strings.Contains(rendered, "kind: Deployment") {
		t.Error("rendered output missing Deployment")
	}
	if !strings.Contains(rendered, "kind: Service") {
		t.Error("rendered output missing Service")
	}
}

func TestIntegrationWordPressCompose(t *testing.T) {
	compose, err := parser.ParseComposeFile("../testdata/wordpress-compose.yml")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	result, err := converter.Convert(compose, defaultOpts())
	if err != nil {
		t.Fatalf("convert error: %v", err)
	}

	// 2 services: wordpress, mysql
	if len(result.Deployments) != 2 {
		t.Errorf("expected 2 deployments, got %d", len(result.Deployments))
	}

	// WordPress should have Service (port 80), mysql should not (no published ports)
	if len(result.Services) != 1 {
		t.Errorf("expected 1 service, got %d", len(result.Services))
	}

	// Both should have ConfigMaps (env vars)
	if len(result.ConfigMaps) < 1 {
		t.Error("expected at least 1 configmap")
	}

	// Both should have PVCs (volumes)
	if len(result.PVCs) != 2 {
		t.Errorf("expected 2 PVCs, got %d", len(result.PVCs))
	}

	// WordPress has HTTP port → Ingress
	if len(result.Ingresses) != 1 {
		t.Errorf("expected 1 ingress, got %d", len(result.Ingresses))
	}

	// ServiceAccounts for each
	if len(result.ServiceAccounts) != 2 {
		t.Errorf("expected 2 service accounts, got %d", len(result.ServiceAccounts))
	}

	// Validate
	vErrors := validator.ValidateManifests(result)
	if validator.HasErrors(vErrors) {
		t.Errorf("validation errors: %v", vErrors)
	}
}

func TestIntegrationFullCompose(t *testing.T) {
	compose, err := parser.ParseComposeFile("../testdata/full-compose.yml")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	result, err := converter.Convert(compose, defaultOpts())
	if err != nil {
		t.Fatalf("convert error: %v", err)
	}

	// 4 services: api, cache, db, web
	if len(result.Deployments) != 4 {
		t.Errorf("expected 4 deployments, got %d", len(result.Deployments))
	}

	// NetworkPolicies for all services (enabled)
	if len(result.NetworkPolicies) != 4 {
		t.Errorf("expected 4 network policies, got %d", len(result.NetworkPolicies))
	}

	// Validate
	vErrors := validator.ValidateManifests(result)
	if validator.HasErrors(vErrors) {
		t.Errorf("validation errors: %v", vErrors)
	}

	// Render should produce valid YAML
	rendered, err := output.RenderManifests(result)
	if err != nil {
		t.Fatalf("render error: %v", err)
	}
	if len(rendered) == 0 {
		t.Error("rendered output is empty")
	}
}

func TestIntegrationDjangoCompose(t *testing.T) {
	compose, err := parser.ParseComposeFile("../testdata/django-compose.yml")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	result, err := converter.Convert(compose, defaultOpts())
	if err != nil {
		t.Fatalf("convert error: %v", err)
	}

	// 4 services: web, celery, db, redis
	if len(result.Deployments) != 4 {
		t.Errorf("expected 4 deployments, got %d", len(result.Deployments))
	}

	// web has deploy.replicas=2 in compose
	for _, d := range result.Deployments {
		if d.Name == "web" {
			if *d.Spec.Replicas != 2 {
				t.Errorf("web replicas: expected 2, got %d", *d.Spec.Replicas)
			}
		}
	}

	// Validate
	vErrors := validator.ValidateManifests(result)
	if validator.HasErrors(vErrors) {
		t.Errorf("validation errors: %v", vErrors)
	}
}

func TestIntegrationMicroservicesCompose(t *testing.T) {
	compose, err := parser.ParseComposeFile("../testdata/microservices-compose.yml")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	result, err := converter.Convert(compose, defaultOpts())
	if err != nil {
		t.Fatalf("convert error: %v", err)
	}

	// 8 services
	if len(result.Deployments) != 8 {
		t.Errorf("expected 8 deployments, got %d", len(result.Deployments))
	}

	// ServiceAccounts for each
	if len(result.ServiceAccounts) != 8 {
		t.Errorf("expected 8 service accounts, got %d", len(result.ServiceAccounts))
	}

	// orders-api has 3 replicas → should have PDB
	foundPDB := false
	for _, pdb := range result.PDBs {
		if pdb.Name == "orders-api" {
			foundPDB = true
			break
		}
	}
	if !foundPDB {
		t.Error("expected PDB for orders-api (3 replicas)")
	}

	// Validate
	vErrors := validator.ValidateManifests(result)
	if validator.HasErrors(vErrors) {
		t.Errorf("validation errors: %v", vErrors)
	}
}

func TestIntegrationNoProbes(t *testing.T) {
	compose, err := parser.ParseComposeFile("../testdata/simple-compose.yml")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	opts := defaultOpts()
	opts.AddProbes = false
	result, err := converter.Convert(compose, opts)
	if err != nil {
		t.Fatalf("convert error: %v", err)
	}

	for _, d := range result.Deployments {
		container := d.Spec.Template.Spec.Containers[0]
		if container.LivenessProbe != nil {
			t.Errorf("service %s should have no liveness probe", d.Name)
		}
		if container.ReadinessProbe != nil {
			t.Errorf("service %s should have no readiness probe", d.Name)
		}
	}
}

func TestIntegrationRenderManifestsYAML(t *testing.T) {
	compose, err := parser.ParseComposeFile("../testdata/simple-compose.yml")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	result, err := converter.Convert(compose, defaultOpts())
	if err != nil {
		t.Fatalf("convert error: %v", err)
	}

	rendered, err := output.RenderManifests(result)
	if err != nil {
		t.Fatalf("render error: %v", err)
	}

	// Should contain YAML document separator
	if !strings.Contains(rendered, "---") {
		t.Error("rendered output missing YAML document separator")
	}

	// Should contain generated header
	if !strings.Contains(rendered, "Generated by kompoze") {
		t.Error("rendered output missing header")
	}

	// Should have valid apiVersion fields
	if !strings.Contains(rendered, "apiVersion:") {
		t.Error("rendered output missing apiVersion")
	}
}
