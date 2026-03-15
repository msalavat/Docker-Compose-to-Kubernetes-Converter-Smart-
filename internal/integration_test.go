package internal_test

import (
	"strings"
	"testing"

	"github.com/compositor/kompoze/internal/converter"
	"github.com/compositor/kompoze/internal/output"
	"github.com/compositor/kompoze/internal/parser"
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

	// web (nginx) -> Deployment, cache (redis) -> Deployment (cache, not database)
	if len(result.Deployments) != 2 {
		t.Errorf("expected 2 deployments, got %d", len(result.Deployments))
	}

	if len(result.Services) != 2 {
		t.Errorf("expected 2 services, got %d", len(result.Services))
	}

	vErrors := validator.ValidateManifests(result)
	if validator.HasErrors(vErrors) {
		t.Errorf("validation errors: %v", vErrors)
	}

	rendered, err := output.RenderManifests(result)
	if err != nil {
		t.Fatalf("render error: %v", err)
	}
	if !strings.Contains(rendered, "kind: Deployment") {
		t.Error("rendered output missing Deployment")
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

	// wordpress -> Deployment, mysql -> StatefulSet
	if len(result.Deployments) != 1 {
		t.Errorf("expected 1 deployment (wordpress), got %d", len(result.Deployments))
	}
	if len(result.StatefulSets) != 1 {
		t.Errorf("expected 1 statefulset (mysql), got %d", len(result.StatefulSets))
	}

	if len(result.ServiceAccounts) != 2 {
		t.Errorf("expected 2 service accounts, got %d", len(result.ServiceAccounts))
	}

	if len(result.Secrets) < 1 {
		t.Error("expected at least 1 secret")
	}

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

	// api, web, cache -> Deployment; db (postgres) -> StatefulSet
	totalWorkloads := len(result.Deployments) + len(result.StatefulSets)
	if totalWorkloads != 4 {
		t.Errorf("expected 4 total workloads, got %d (dep=%d, ss=%d)",
			totalWorkloads, len(result.Deployments), len(result.StatefulSets))
	}

	if len(result.NetworkPolicies) != 4 {
		t.Errorf("expected 4 network policies, got %d", len(result.NetworkPolicies))
	}

	vErrors := validator.ValidateManifests(result)
	if validator.HasErrors(vErrors) {
		t.Errorf("validation errors: %v", vErrors)
	}

	rendered, err := output.RenderManifests(result)
	if err != nil {
		t.Fatalf("render error: %v", err)
	}
	if !strings.Contains(rendered, "kind: StatefulSet") {
		t.Error("rendered output missing StatefulSet")
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

	// web, celery, redis -> Deployment; db (postgres) -> StatefulSet
	if len(result.Deployments) != 3 {
		t.Errorf("expected 3 deployments, got %d", len(result.Deployments))
	}
	if len(result.StatefulSets) != 1 {
		t.Errorf("expected 1 statefulset (db), got %d", len(result.StatefulSets))
	}

	for _, d := range result.Deployments {
		if d.Name == "web" {
			if *d.Spec.Replicas != 2 {
				t.Errorf("web replicas: expected 2, got %d", *d.Spec.Replicas)
			}
		}
	}

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

	// gateway, users-api, orders-api, products-api, cache -> 5 Deployments
	// users-db, orders-db, products-db -> 3 StatefulSets
	if len(result.Deployments) != 5 {
		t.Errorf("expected 5 deployments, got %d", len(result.Deployments))
	}
	if len(result.StatefulSets) != 3 {
		t.Errorf("expected 3 statefulsets, got %d", len(result.StatefulSets))
	}

	if len(result.ServiceAccounts) != 8 {
		t.Errorf("expected 8 service accounts, got %d", len(result.ServiceAccounts))
	}

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
			t.Errorf("deployment %s should have no liveness probe", d.Name)
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

	if !strings.Contains(rendered, "---") {
		t.Error("rendered output missing YAML document separator")
	}
	if !strings.Contains(rendered, "Generated by kompoze") {
		t.Error("rendered output missing header")
	}
}

func TestIntegrationSecretGeneration(t *testing.T) {
	compose, err := parser.ParseComposeFile("../testdata/wordpress-compose.yml")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	result, err := converter.Convert(compose, defaultOpts())
	if err != nil {
		t.Fatalf("convert error: %v", err)
	}

	if len(result.Secrets) < 1 {
		t.Fatal("expected at least 1 secret")
	}

	for _, s := range result.Secrets {
		if s.Name == "mysql-secret" {
			if _, ok := s.StringData["MYSQL_PASSWORD"]; !ok {
				t.Error("mysql secret missing MYSQL_PASSWORD")
			}
		}
	}

	rendered, err := output.RenderManifests(result)
	if err != nil {
		t.Fatalf("render error: %v", err)
	}
	if !strings.Contains(rendered, "kind: Secret") {
		t.Error("rendered output missing Secret")
	}
}

func TestIntegrationStatefulSet(t *testing.T) {
	compose, err := parser.ParseComposeFile("../testdata/wordpress-compose.yml")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	result, err := converter.Convert(compose, defaultOpts())
	if err != nil {
		t.Fatalf("convert error: %v", err)
	}

	if len(result.StatefulSets) != 1 {
		t.Fatalf("expected 1 statefulset, got %d", len(result.StatefulSets))
	}

	ss := result.StatefulSets[0]
	if ss.Name != "mysql" {
		t.Errorf("expected name=mysql, got %s", ss.Name)
	}
	if ss.Kind != "StatefulSet" {
		t.Errorf("expected Kind=StatefulSet, got %s", ss.Kind)
	}
	if ss.Spec.ServiceName != "mysql" {
		t.Errorf("expected serviceName=mysql, got %s", ss.Spec.ServiceName)
	}
}
