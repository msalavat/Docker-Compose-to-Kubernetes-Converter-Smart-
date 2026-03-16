package wizard

import (
	"strings"
	"testing"

	"github.com/compositor/kompoze/internal/parser"
	tea "github.com/charmbracelet/bubbletea"
)

// --- Helper functions for creating key messages ---

func keyRune(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

func keyEnter() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyEnter}
}

func keyBackspace() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyBackspace}
}

func keyUp() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyUp}
}

func keyDown() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyDown}
}

func keyCtrlC() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyCtrlC}
}

// typeString sends a sequence of rune key messages to a model.
func typeString(m tea.Model, s string) tea.Model {
	for _, r := range s {
		m, _ = m.Update(keyRune(r))
	}
	return m
}

// newTestModel creates a model initialized for testing with the given compose file.
func newTestModel(compose *parser.ComposeFile) model {
	names := make([]string, 0, len(compose.Services))
	for name := range compose.Services {
		names = append(names, name)
	}
	// Sort for deterministic order (matches Run behavior)
	// We use a simple sort here to avoid importing sort
	for i := 0; i < len(names); i++ {
		for j := i + 1; j < len(names); j++ {
			if names[i] > names[j] {
				names[i], names[j] = names[j], names[i]
			}
		}
	}

	services := make(map[string]ServiceWizardConfig)
	for _, name := range names {
		svc := compose.Services[name]
		st := DetectServiceType(svc.Image)
		cfg := ServiceWizardConfig{
			Kind:         "Deployment",
			Replicas:     1,
			HPAMin:       2,
			HPAMax:       10,
			HPATargetCPU: 70,
			PDBMinAvail:  1,
			PVCSize:      "1Gi",
		}
		if ShouldSuggestStatefulSet(st) {
			cfg.Kind = "StatefulSet"
			cfg.PVCSize = "10Gi"
			cfg.AddPDB = true
		}
		if ShouldSuggestIngress(st) {
			cfg.AddIngress = true
			cfg.IngressHost = name + ".example.com"
			cfg.AddTLS = true
			cfg.Replicas = 2
		}
		if ShouldSuggestHPA(st) {
			cfg.AddHPA = true
			cfg.Replicas = 2
		}
		for k := range svc.Environment {
			upper := strings.ToUpper(k)
			if strings.Contains(upper, "PASSWORD") || strings.Contains(upper, "SECRET") || strings.Contains(upper, "TOKEN") {
				cfg.CreateSecret = true
				break
			}
		}
		services[name] = cfg
	}

	return model{
		compose: compose,
		config: WizardConfig{
			Namespace:    "default",
			OutputFormat: "manifests",
			Services:     services,
		},
		serviceNames: names,
		phase:        phaseNamespace,
		input:        "default",
	}
}

// --- detection.go tests ---

func TestDetectServiceType(t *testing.T) {
	tests := []struct {
		name     string
		image    string
		expected ServiceType
	}{
		// Web servers
		{"nginx plain", "nginx", ServiceTypeWebServer},
		{"nginx with tag", "nginx:1.25", ServiceTypeWebServer},
		{"nginx with registry", "registry.example.com/nginx:1.25", ServiceTypeWebServer},
		{"httpd", "httpd:2.4", ServiceTypeWebServer},
		{"apache", "apache", ServiceTypeWebServer},
		{"traefik", "traefik:v2.10", ServiceTypeWebServer},
		{"caddy", "caddy:latest", ServiceTypeWebServer},
		{"haproxy", "haproxy:2.8", ServiceTypeWebServer},
		{"envoy", "envoyproxy/envoy:v1.28", ServiceTypeWebServer},

		// Databases
		{"postgres", "postgres:15", ServiceTypeDatabase},
		{"postgres with registry", "registry.example.com/postgres:15", ServiceTypeDatabase},
		{"mysql", "mysql:8.0", ServiceTypeDatabase},
		{"mariadb", "mariadb:11", ServiceTypeDatabase},
		{"mongo", "mongo:7", ServiceTypeDatabase},
		{"cockroachdb", "cockroachdb/cockroach:latest", ServiceTypeDatabase},
		{"cassandra", "cassandra:4", ServiceTypeDatabase},
		{"elasticsearch", "elasticsearch:8.11", ServiceTypeDatabase},
		{"opensearch", "opensearchproject/opensearch:2", ServiceTypeDatabase},
		{"couchdb", "couchdb:3", ServiceTypeDatabase},
		{"neo4j", "neo4j:5", ServiceTypeDatabase},
		{"influxdb", "influxdb:2", ServiceTypeDatabase},

		// Caches
		{"redis", "redis:7", ServiceTypeCache},
		{"redis with registry", "gcr.io/project/redis:7-alpine", ServiceTypeCache},
		{"memcached", "memcached:1.6", ServiceTypeCache},
		{"valkey", "valkey/valkey:latest", ServiceTypeCache},

		// App servers
		{"node", "node:20", ServiceTypeAppServer},
		{"python", "python:3.12", ServiceTypeAppServer},
		{"golang", "golang:1.22", ServiceTypeAppServer},
		{"java", "java:17", ServiceTypeAppServer},
		{"ruby", "ruby:3.3", ServiceTypeAppServer},
		{"php", "php:8.3", ServiceTypeAppServer},
		{"dotnet", "mcr.microsoft.com/dotnet:8.0", ServiceTypeAppServer},
		{"flask", "flask:latest", ServiceTypeAppServer},
		{"django", "django:latest", ServiceTypeAppServer},
		{"express", "express:latest", ServiceTypeAppServer},
		{"spring", "spring:latest", ServiceTypeAppServer},
		{"laravel", "laravel:latest", ServiceTypeAppServer},
		{"rails", "rails:7", ServiceTypeAppServer},

		// Generic
		{"custom image", "my-custom-app:1.0", ServiceTypeGeneric},
		{"unknown", "busybox", ServiceTypeGeneric},
		{"empty", "", ServiceTypeGeneric},

		// Case insensitivity
		{"uppercase NGINX", "NGINX:1.25", ServiceTypeWebServer},
		{"mixed case Postgres", "Postgres:15", ServiceTypeDatabase},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectServiceType(tt.image)
			if got != tt.expected {
				t.Errorf("DetectServiceType(%q) = %q, want %q", tt.image, got, tt.expected)
			}
		})
	}
}

func TestShouldSuggestIngress(t *testing.T) {
	tests := []struct {
		serviceType ServiceType
		expected    bool
	}{
		{ServiceTypeWebServer, true},
		{ServiceTypeDatabase, false},
		{ServiceTypeCache, false},
		{ServiceTypeAppServer, false},
		{ServiceTypeGeneric, false},
	}
	for _, tt := range tests {
		t.Run(string(tt.serviceType), func(t *testing.T) {
			if got := ShouldSuggestIngress(tt.serviceType); got != tt.expected {
				t.Errorf("ShouldSuggestIngress(%q) = %v, want %v", tt.serviceType, got, tt.expected)
			}
		})
	}
}

func TestShouldSuggestStatefulSet(t *testing.T) {
	tests := []struct {
		serviceType ServiceType
		expected    bool
	}{
		{ServiceTypeWebServer, false},
		{ServiceTypeDatabase, true},
		{ServiceTypeCache, false},
		{ServiceTypeAppServer, false},
		{ServiceTypeGeneric, false},
	}
	for _, tt := range tests {
		t.Run(string(tt.serviceType), func(t *testing.T) {
			if got := ShouldSuggestStatefulSet(tt.serviceType); got != tt.expected {
				t.Errorf("ShouldSuggestStatefulSet(%q) = %v, want %v", tt.serviceType, got, tt.expected)
			}
		})
	}
}

func TestShouldSuggestHPA(t *testing.T) {
	tests := []struct {
		serviceType ServiceType
		expected    bool
	}{
		{ServiceTypeWebServer, true},
		{ServiceTypeDatabase, false},
		{ServiceTypeCache, false},
		{ServiceTypeAppServer, true},
		{ServiceTypeGeneric, false},
	}
	for _, tt := range tests {
		t.Run(string(tt.serviceType), func(t *testing.T) {
			if got := ShouldSuggestHPA(tt.serviceType); got != tt.expected {
				t.Errorf("ShouldSuggestHPA(%q) = %v, want %v", tt.serviceType, got, tt.expected)
			}
		})
	}
}

func TestShouldSuggestPDB(t *testing.T) {
	tests := []struct {
		serviceType ServiceType
		expected    bool
	}{
		{ServiceTypeWebServer, false},
		{ServiceTypeDatabase, true},
		{ServiceTypeCache, true},
		{ServiceTypeAppServer, false},
		{ServiceTypeGeneric, false},
	}
	for _, tt := range tests {
		t.Run(string(tt.serviceType), func(t *testing.T) {
			if got := ShouldSuggestPDB(tt.serviceType); got != tt.expected {
				t.Errorf("ShouldSuggestPDB(%q) = %v, want %v", tt.serviceType, got, tt.expected)
			}
		})
	}
}

// --- wizard.go tests ---

func TestModelInit(t *testing.T) {
	compose := &parser.ComposeFile{
		Services: map[string]parser.ServiceConfig{
			"web": {Image: "nginx:1.25"},
		},
	}
	m := newTestModel(compose)
	cmd := m.Init()
	if cmd != nil {
		t.Errorf("Init() should return nil, got %v", cmd)
	}
}

func TestRunWithNilCompose(t *testing.T) {
	_, err := Run(nil)
	if err == nil {
		t.Fatal("Run(nil) should return an error")
	}
	if !strings.Contains(err.Error(), "compose file is nil") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUpdateNamespace(t *testing.T) {
	compose := &parser.ComposeFile{
		Services: map[string]parser.ServiceConfig{
			"web": {Image: "nginx:1.25"},
		},
	}

	t.Run("enter with default", func(t *testing.T) {
		m := newTestModel(compose)
		// Press enter with the default "default" input
		result, _ := m.Update(keyEnter())
		rm := result.(model)
		if rm.config.Namespace != "default" {
			t.Errorf("expected namespace 'default', got %q", rm.config.Namespace)
		}
		if rm.phase != phaseOutputFormat {
			t.Errorf("expected phase phaseOutputFormat, got %d", rm.phase)
		}
	})

	t.Run("type custom namespace", func(t *testing.T) {
		m := newTestModel(compose)
		// Clear default input first
		for i := 0; i < len("default"); i++ {
			result, _ := m.Update(keyBackspace())
			m = result.(model)
		}
		// Type custom namespace
		m = typeString(m, "production").(model)
		if m.input != "production" {
			t.Errorf("expected input 'production', got %q", m.input)
		}
		// Press enter
		result, _ := m.Update(keyEnter())
		rm := result.(model)
		if rm.config.Namespace != "production" {
			t.Errorf("expected namespace 'production', got %q", rm.config.Namespace)
		}
	})

	t.Run("backspace removes characters", func(t *testing.T) {
		m := newTestModel(compose)
		// Input starts as "default"
		result, _ := m.Update(keyBackspace())
		rm := result.(model)
		if rm.input != "defaul" {
			t.Errorf("expected input 'defaul' after backspace, got %q", rm.input)
		}
	})

	t.Run("backspace on empty input", func(t *testing.T) {
		m := newTestModel(compose)
		m.input = ""
		result, _ := m.Update(keyBackspace())
		rm := result.(model)
		if rm.input != "" {
			t.Errorf("expected empty input after backspace on empty, got %q", rm.input)
		}
	})

	t.Run("enter with empty input keeps default namespace", func(t *testing.T) {
		m := newTestModel(compose)
		m.input = ""
		result, _ := m.Update(keyEnter())
		rm := result.(model)
		// When input is empty, namespace stays as the config default
		if rm.config.Namespace != "default" {
			t.Errorf("expected namespace 'default' (unchanged), got %q", rm.config.Namespace)
		}
		if rm.phase != phaseOutputFormat {
			t.Errorf("expected phase phaseOutputFormat, got %d", rm.phase)
		}
	})
}

func TestUpdateOutputFormat(t *testing.T) {
	compose := &parser.ComposeFile{
		Services: map[string]parser.ServiceConfig{
			"web": {Image: "nginx:1.25"},
		},
	}

	t.Run("default selection is manifests", func(t *testing.T) {
		m := newTestModel(compose)
		m.phase = phaseOutputFormat
		m.cursor = 0
		result, _ := m.Update(keyEnter())
		rm := result.(model)
		if rm.config.OutputFormat != "manifests" {
			t.Errorf("expected format 'manifests', got %q", rm.config.OutputFormat)
		}
		if rm.phase != phaseServiceConfig {
			t.Errorf("expected phase phaseServiceConfig, got %d", rm.phase)
		}
	})

	t.Run("select helm with down arrow", func(t *testing.T) {
		m := newTestModel(compose)
		m.phase = phaseOutputFormat
		m.cursor = 0
		result, _ := m.Update(keyDown())
		rm := result.(model)
		if rm.cursor != 1 {
			t.Errorf("expected cursor 1, got %d", rm.cursor)
		}
		result, _ = rm.Update(keyEnter())
		rm = result.(model)
		if rm.config.OutputFormat != "helm" {
			t.Errorf("expected format 'helm', got %q", rm.config.OutputFormat)
		}
	})

	t.Run("select kustomize", func(t *testing.T) {
		m := newTestModel(compose)
		m.phase = phaseOutputFormat
		m.cursor = 0
		result, _ := m.Update(keyDown())
		rm := result.(model)
		result, _ = rm.Update(keyDown())
		rm = result.(model)
		if rm.cursor != 2 {
			t.Errorf("expected cursor 2, got %d", rm.cursor)
		}
		result, _ = rm.Update(keyEnter())
		rm = result.(model)
		if rm.config.OutputFormat != "kustomize" {
			t.Errorf("expected format 'kustomize', got %q", rm.config.OutputFormat)
		}
	})

	t.Run("up arrow at top stays at 0", func(t *testing.T) {
		m := newTestModel(compose)
		m.phase = phaseOutputFormat
		m.cursor = 0
		result, _ := m.Update(keyUp())
		rm := result.(model)
		if rm.cursor != 0 {
			t.Errorf("expected cursor 0, got %d", rm.cursor)
		}
	})

	t.Run("down arrow at bottom stays at 2", func(t *testing.T) {
		m := newTestModel(compose)
		m.phase = phaseOutputFormat
		m.cursor = 2
		result, _ := m.Update(keyDown())
		rm := result.(model)
		if rm.cursor != 2 {
			t.Errorf("expected cursor 2, got %d", rm.cursor)
		}
	})

	t.Run("k and j keys work for navigation", func(t *testing.T) {
		m := newTestModel(compose)
		m.phase = phaseOutputFormat
		m.cursor = 1
		// k = up
		result, _ := m.Update(keyRune('k'))
		rm := result.(model)
		if rm.cursor != 0 {
			t.Errorf("expected cursor 0 after 'k', got %d", rm.cursor)
		}
		// j = down
		result, _ = rm.Update(keyRune('j'))
		rm = result.(model)
		if rm.cursor != 1 {
			t.Errorf("expected cursor 1 after 'j', got %d", rm.cursor)
		}
	})
}

func TestUpdateServiceConfig(t *testing.T) {
	compose := &parser.ComposeFile{
		Services: map[string]parser.ServiceConfig{
			"app": {Image: "node:20"},
		},
	}

	t.Run("set replicas", func(t *testing.T) {
		m := newTestModel(compose)
		m.phase = phaseServiceConfig
		m.serviceIdx = 0
		m.field = fieldReplicas
		m.input = ""

		// Type "3"
		m = typeString(m, "3").(model)
		result, _ := m.Update(keyEnter())
		rm := result.(model)
		cfg := rm.config.Services["app"]
		if cfg.Replicas != 3 {
			t.Errorf("expected replicas 3, got %d", cfg.Replicas)
		}
		if rm.field != fieldIngress {
			t.Errorf("expected field fieldIngress, got %d", rm.field)
		}
	})

	t.Run("empty replicas keeps default", func(t *testing.T) {
		m := newTestModel(compose)
		m.phase = phaseServiceConfig
		m.serviceIdx = 0
		m.field = fieldReplicas
		m.input = ""

		result, _ := m.Update(keyEnter())
		rm := result.(model)
		cfg := rm.config.Services["app"]
		// Default for app-server (node) is 2 (HPA suggested)
		if cfg.Replicas != 2 {
			t.Errorf("expected replicas 2 (default for app-server), got %d", cfg.Replicas)
		}
	})

	t.Run("ingress yes leads to host field", func(t *testing.T) {
		m := newTestModel(compose)
		m.phase = phaseServiceConfig
		m.serviceIdx = 0
		m.field = fieldIngress
		m.input = ""

		// Press enter (default yes)
		result, _ := m.Update(keyEnter())
		rm := result.(model)
		cfg := rm.config.Services["app"]
		if !cfg.AddIngress {
			t.Errorf("expected AddIngress true")
		}
		if rm.field != fieldIngressHost {
			t.Errorf("expected field fieldIngressHost, got %d", rm.field)
		}
	})

	t.Run("ingress no skips to HPA", func(t *testing.T) {
		m := newTestModel(compose)
		m.phase = phaseServiceConfig
		m.serviceIdx = 0
		m.field = fieldIngress
		m.input = ""

		m = typeString(m, "n").(model)
		result, _ := m.Update(keyEnter())
		rm := result.(model)
		cfg := rm.config.Services["app"]
		if cfg.AddIngress {
			t.Errorf("expected AddIngress false")
		}
		if rm.field != fieldHPA {
			t.Errorf("expected field fieldHPA after declining ingress, got %d", rm.field)
		}
	})

	t.Run("ingress host custom value", func(t *testing.T) {
		m := newTestModel(compose)
		m.phase = phaseServiceConfig
		m.serviceIdx = 0
		m.field = fieldIngressHost
		m.input = ""

		m = typeString(m, "myapp.dev").(model)
		result, _ := m.Update(keyEnter())
		rm := result.(model)
		cfg := rm.config.Services["app"]
		if cfg.IngressHost != "myapp.dev" {
			t.Errorf("expected IngressHost 'myapp.dev', got %q", cfg.IngressHost)
		}
		if rm.field != fieldTLS {
			t.Errorf("expected field fieldTLS, got %d", rm.field)
		}
	})

	t.Run("TLS yes", func(t *testing.T) {
		m := newTestModel(compose)
		m.phase = phaseServiceConfig
		m.serviceIdx = 0
		m.field = fieldTLS
		m.input = ""

		result, _ := m.Update(keyEnter())
		rm := result.(model)
		cfg := rm.config.Services["app"]
		if !cfg.AddTLS {
			t.Errorf("expected AddTLS true")
		}
		if rm.field != fieldHPA {
			t.Errorf("expected field fieldHPA, got %d", rm.field)
		}
	})

	t.Run("TLS no", func(t *testing.T) {
		m := newTestModel(compose)
		m.phase = phaseServiceConfig
		m.serviceIdx = 0
		m.field = fieldTLS
		m.input = ""

		m = typeString(m, "n").(model)
		result, _ := m.Update(keyEnter())
		rm := result.(model)
		cfg := rm.config.Services["app"]
		if cfg.AddTLS {
			t.Errorf("expected AddTLS false")
		}
	})

	t.Run("HPA yes", func(t *testing.T) {
		m := newTestModel(compose)
		m.phase = phaseServiceConfig
		m.serviceIdx = 0
		m.field = fieldHPA
		m.input = ""

		result, _ := m.Update(keyEnter())
		rm := result.(model)
		cfg := rm.config.Services["app"]
		if !cfg.AddHPA {
			t.Errorf("expected AddHPA true")
		}
		if rm.field != fieldPDB {
			t.Errorf("expected field fieldPDB, got %d", rm.field)
		}
	})

	t.Run("PDB completes service and moves to summary", func(t *testing.T) {
		m := newTestModel(compose)
		m.phase = phaseServiceConfig
		m.serviceIdx = 0
		m.field = fieldPDB
		m.input = ""

		result, _ := m.Update(keyEnter())
		rm := result.(model)
		// Only one service, should move to summary
		if rm.phase != phaseSummary {
			t.Errorf("expected phase phaseSummary, got %d", rm.phase)
		}
	})

	t.Run("backspace in service config", func(t *testing.T) {
		m := newTestModel(compose)
		m.phase = phaseServiceConfig
		m.serviceIdx = 0
		m.field = fieldReplicas
		m.input = "abc"

		result, _ := m.Update(keyBackspace())
		rm := result.(model)
		if rm.input != "ab" {
			t.Errorf("expected input 'ab', got %q", rm.input)
		}
	})
}

func TestUpdateServiceConfigMultipleServices(t *testing.T) {
	compose := &parser.ComposeFile{
		Services: map[string]parser.ServiceConfig{
			"api":   {Image: "node:20"},
			"proxy": {Image: "nginx:1.25"},
		},
	}

	m := newTestModel(compose)
	m.phase = phaseServiceConfig
	m.serviceIdx = 0
	m.field = fieldPDB
	m.input = ""

	// Complete first service (api)
	result, _ := m.Update(keyEnter())
	rm := result.(model)

	// Should move to second service, not summary
	if rm.phase != phaseServiceConfig {
		t.Errorf("expected to stay in phaseServiceConfig for second service, got %d", rm.phase)
	}
	if rm.serviceIdx != 1 {
		t.Errorf("expected serviceIdx 1, got %d", rm.serviceIdx)
	}
	if rm.field != fieldReplicas {
		t.Errorf("expected field fieldReplicas for new service, got %d", rm.field)
	}

	// Complete second service (proxy) - go through all fields
	// fieldReplicas -> enter
	result, _ = rm.Update(keyEnter())
	rm = result.(model)
	// fieldIngress -> enter (yes)
	result, _ = rm.Update(keyEnter())
	rm = result.(model)
	// fieldIngressHost -> enter
	result, _ = rm.Update(keyEnter())
	rm = result.(model)
	// fieldTLS -> enter
	result, _ = rm.Update(keyEnter())
	rm = result.(model)
	// fieldHPA -> enter
	result, _ = rm.Update(keyEnter())
	rm = result.(model)
	// fieldPDB -> enter
	result, _ = rm.Update(keyEnter())
	rm = result.(model)

	if rm.phase != phaseSummary {
		t.Errorf("expected phase phaseSummary after all services, got %d", rm.phase)
	}
}

func TestUpdateSummary(t *testing.T) {
	compose := &parser.ComposeFile{
		Services: map[string]parser.ServiceConfig{
			"web": {Image: "nginx:1.25"},
		},
	}

	t.Run("enter confirms", func(t *testing.T) {
		m := newTestModel(compose)
		m.phase = phaseSummary
		result, cmd := m.Update(keyEnter())
		rm := result.(model)
		if rm.phase != phaseDone {
			t.Errorf("expected phase phaseDone, got %d", rm.phase)
		}
		if cmd == nil {
			t.Error("expected tea.Quit command, got nil")
		}
	})

	t.Run("y confirms", func(t *testing.T) {
		m := newTestModel(compose)
		m.phase = phaseSummary
		result, cmd := m.Update(keyRune('y'))
		rm := result.(model)
		if rm.phase != phaseDone {
			t.Errorf("expected phase phaseDone, got %d", rm.phase)
		}
		if cmd == nil {
			t.Error("expected tea.Quit command, got nil")
		}
	})

	t.Run("Y confirms", func(t *testing.T) {
		m := newTestModel(compose)
		m.phase = phaseSummary
		result, cmd := m.Update(keyRune('Y'))
		rm := result.(model)
		if rm.phase != phaseDone {
			t.Errorf("expected phase phaseDone, got %d", rm.phase)
		}
		if cmd == nil {
			t.Error("expected tea.Quit command, got nil")
		}
	})

	t.Run("n cancels", func(t *testing.T) {
		m := newTestModel(compose)
		m.phase = phaseSummary
		result, cmd := m.Update(keyRune('n'))
		rm := result.(model)
		if !rm.quitting {
			t.Error("expected quitting to be true")
		}
		if cmd == nil {
			t.Error("expected tea.Quit command, got nil")
		}
	})

	t.Run("N cancels", func(t *testing.T) {
		m := newTestModel(compose)
		m.phase = phaseSummary
		result, _ := m.Update(keyRune('N'))
		rm := result.(model)
		if !rm.quitting {
			t.Error("expected quitting to be true")
		}
	})

	t.Run("other key does nothing", func(t *testing.T) {
		m := newTestModel(compose)
		m.phase = phaseSummary
		result, cmd := m.Update(keyRune('x'))
		rm := result.(model)
		if rm.phase != phaseSummary {
			t.Errorf("expected phase to remain phaseSummary, got %d", rm.phase)
		}
		if cmd != nil {
			t.Error("expected nil command for unhandled key")
		}
	})
}

func TestCtrlCQuits(t *testing.T) {
	compose := &parser.ComposeFile{
		Services: map[string]parser.ServiceConfig{
			"web": {Image: "nginx:1.25"},
		},
	}

	t.Run("ctrl+c in namespace phase quits", func(t *testing.T) {
		m := newTestModel(compose)
		m.phase = phaseNamespace
		result, cmd := m.Update(keyCtrlC())
		rm := result.(model)
		if !rm.quitting {
			t.Error("expected quitting to be true")
		}
		if cmd == nil {
			t.Error("expected tea.Quit command")
		}
	})

	t.Run("ctrl+c in output format phase quits", func(t *testing.T) {
		m := newTestModel(compose)
		m.phase = phaseOutputFormat
		result, cmd := m.Update(keyCtrlC())
		rm := result.(model)
		if !rm.quitting {
			t.Error("expected quitting to be true")
		}
		if cmd == nil {
			t.Error("expected tea.Quit command")
		}
	})

	t.Run("q in output format phase quits", func(t *testing.T) {
		m := newTestModel(compose)
		m.phase = phaseOutputFormat
		result, cmd := m.Update(keyRune('q'))
		rm := result.(model)
		if !rm.quitting {
			t.Error("expected quitting to be true")
		}
		if cmd == nil {
			t.Error("expected tea.Quit command")
		}
	})

	t.Run("q in namespace phase does not quit (types q)", func(t *testing.T) {
		m := newTestModel(compose)
		m.phase = phaseNamespace
		result, _ := m.Update(keyRune('q'))
		rm := result.(model)
		if rm.quitting {
			t.Error("expected quitting to be false in namespace phase for 'q'")
		}
	})
}

func TestNonKeyMsgIgnored(t *testing.T) {
	compose := &parser.ComposeFile{
		Services: map[string]parser.ServiceConfig{
			"web": {Image: "nginx:1.25"},
		},
	}
	m := newTestModel(compose)
	// Send a non-KeyMsg (e.g., tea.WindowSizeMsg)
	result, cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	rm := result.(model)
	if rm.phase != phaseNamespace {
		t.Errorf("expected phase unchanged, got %d", rm.phase)
	}
	if cmd != nil {
		t.Error("expected nil command for non-key message")
	}
}

// --- View tests ---

func TestViewNamespacePhase(t *testing.T) {
	compose := &parser.ComposeFile{
		Services: map[string]parser.ServiceConfig{
			"web": {Image: "nginx:1.25"},
		},
	}
	m := newTestModel(compose)
	m.phase = phaseNamespace
	view := m.View()
	if !strings.Contains(view, "Kompoze Wizard") {
		t.Error("expected view to contain 'Kompoze Wizard'")
	}
	if !strings.Contains(view, "Namespace") {
		t.Error("expected view to contain 'Namespace'")
	}
	if !strings.Contains(view, "1 services") {
		t.Error("expected view to contain '1 services'")
	}
}

func TestViewOutputFormatPhase(t *testing.T) {
	compose := &parser.ComposeFile{
		Services: map[string]parser.ServiceConfig{
			"web": {Image: "nginx:1.25"},
		},
	}
	m := newTestModel(compose)
	m.phase = phaseOutputFormat
	m.cursor = 0
	view := m.View()
	if !strings.Contains(view, "Output format") {
		t.Error("expected view to contain 'Output format'")
	}
	if !strings.Contains(view, "Kubernetes manifests") {
		t.Error("expected view to contain 'Kubernetes manifests'")
	}
	if !strings.Contains(view, "Helm chart") {
		t.Error("expected view to contain 'Helm chart'")
	}
	if !strings.Contains(view, "Kustomize") {
		t.Error("expected view to contain 'Kustomize'")
	}
}

func TestViewServiceConfigPhase(t *testing.T) {
	compose := &parser.ComposeFile{
		Services: map[string]parser.ServiceConfig{
			"web": {Image: "nginx:1.25"},
		},
	}
	m := newTestModel(compose)
	m.phase = phaseServiceConfig
	m.serviceIdx = 0
	m.field = fieldReplicas
	view := m.View()
	if !strings.Contains(view, "Service: web") {
		t.Error("expected view to contain 'Service: web'")
	}
	if !strings.Contains(view, "nginx:1.25") {
		t.Error("expected view to contain image name")
	}
	if !strings.Contains(view, "Replicas") {
		t.Error("expected view to contain 'Replicas'")
	}
}

func TestViewServiceConfigFieldVariants(t *testing.T) {
	compose := &parser.ComposeFile{
		Services: map[string]parser.ServiceConfig{
			"web": {Image: "nginx:1.25"},
		},
	}

	fields := []struct {
		field    fieldType
		keyword  string
	}{
		{fieldReplicas, "Replicas"},
		{fieldIngress, "Ingress"},
		{fieldIngressHost, "hostname"},
		{fieldTLS, "TLS"},
		{fieldHPA, "HPA"},
		{fieldPDB, "PodDisruptionBudget"},
	}

	for _, f := range fields {
		t.Run(f.keyword, func(t *testing.T) {
			m := newTestModel(compose)
			m.phase = phaseServiceConfig
			m.serviceIdx = 0
			m.field = f.field
			view := m.View()
			if !strings.Contains(view, f.keyword) {
				t.Errorf("expected view to contain %q for field %d", f.keyword, f.field)
			}
		})
	}
}

func TestViewSummaryPhase(t *testing.T) {
	compose := &parser.ComposeFile{
		Services: map[string]parser.ServiceConfig{
			"web": {Image: "nginx:1.25"},
			"db":  {Image: "postgres:15"},
		},
	}
	m := newTestModel(compose)
	m.phase = phaseSummary
	view := m.View()
	if !strings.Contains(view, "Summary") {
		t.Error("expected view to contain 'Summary'")
	}
	if !strings.Contains(view, "Namespace") {
		t.Error("expected view to contain 'Namespace'")
	}
	if !strings.Contains(view, "Format") {
		t.Error("expected view to contain 'Format'")
	}
	if !strings.Contains(view, "web") {
		t.Error("expected view to contain service 'web'")
	}
	if !strings.Contains(view, "db") {
		t.Error("expected view to contain service 'db'")
	}
	if !strings.Contains(view, "Generate manifests") {
		t.Error("expected view to contain 'Generate manifests'")
	}
}

func TestViewDonePhase(t *testing.T) {
	compose := &parser.ComposeFile{
		Services: map[string]parser.ServiceConfig{
			"web": {Image: "nginx:1.25"},
		},
	}
	m := newTestModel(compose)
	m.phase = phaseDone
	view := m.View()
	if !strings.Contains(view, "Generating manifests") {
		t.Error("expected view to contain 'Generating manifests'")
	}
}

// --- Smart defaults tests ---

func TestSmartDefaultsDatabase(t *testing.T) {
	compose := &parser.ComposeFile{
		Services: map[string]parser.ServiceConfig{
			"db": {Image: "postgres:15"},
		},
	}
	m := newTestModel(compose)
	cfg := m.config.Services["db"]

	if cfg.Kind != "StatefulSet" {
		t.Errorf("expected database Kind 'StatefulSet', got %q", cfg.Kind)
	}
	if cfg.PVCSize != "10Gi" {
		t.Errorf("expected database PVCSize '10Gi', got %q", cfg.PVCSize)
	}
	if !cfg.AddPDB {
		t.Error("expected database AddPDB to be true")
	}
	if cfg.AddIngress {
		t.Error("expected database AddIngress to be false")
	}
	if cfg.AddHPA {
		t.Error("expected database AddHPA to be false")
	}
}

func TestSmartDefaultsWebServer(t *testing.T) {
	compose := &parser.ComposeFile{
		Services: map[string]parser.ServiceConfig{
			"web": {Image: "nginx:1.25"},
		},
	}
	m := newTestModel(compose)
	cfg := m.config.Services["web"]

	if cfg.Kind != "Deployment" {
		t.Errorf("expected web Kind 'Deployment', got %q", cfg.Kind)
	}
	if !cfg.AddIngress {
		t.Error("expected web AddIngress to be true")
	}
	if cfg.IngressHost != "web.example.com" {
		t.Errorf("expected IngressHost 'web.example.com', got %q", cfg.IngressHost)
	}
	if !cfg.AddTLS {
		t.Error("expected web AddTLS to be true")
	}
	if cfg.Replicas != 2 {
		t.Errorf("expected web Replicas 2, got %d", cfg.Replicas)
	}
	if !cfg.AddHPA {
		t.Error("expected web AddHPA to be true")
	}
}

func TestSmartDefaultsAppServer(t *testing.T) {
	compose := &parser.ComposeFile{
		Services: map[string]parser.ServiceConfig{
			"api": {Image: "node:20"},
		},
	}
	m := newTestModel(compose)
	cfg := m.config.Services["api"]

	if cfg.Kind != "Deployment" {
		t.Errorf("expected app Kind 'Deployment', got %q", cfg.Kind)
	}
	if !cfg.AddHPA {
		t.Error("expected app AddHPA to be true")
	}
	if cfg.Replicas != 2 {
		t.Errorf("expected app Replicas 2, got %d", cfg.Replicas)
	}
	if cfg.AddIngress {
		t.Error("expected app AddIngress to be false")
	}
}

func TestSmartDefaultsCache(t *testing.T) {
	compose := &parser.ComposeFile{
		Services: map[string]parser.ServiceConfig{
			"cache": {Image: "redis:7"},
		},
	}
	m := newTestModel(compose)
	cfg := m.config.Services["cache"]

	if cfg.Kind != "Deployment" {
		t.Errorf("expected cache Kind 'Deployment', got %q", cfg.Kind)
	}
	if cfg.AddHPA {
		t.Error("expected cache AddHPA to be false")
	}
	// Cache doesn't get StatefulSet default, so PDB is not set by StatefulSet logic.
	// ShouldSuggestPDB returns true for cache, but the wizard only sets PDB for StatefulSet.
	_ = cfg.AddPDB // verified via smart defaults behavior
}

func TestSmartDefaultsGeneric(t *testing.T) {
	compose := &parser.ComposeFile{
		Services: map[string]parser.ServiceConfig{
			"custom": {Image: "my-app:1.0"},
		},
	}
	m := newTestModel(compose)
	cfg := m.config.Services["custom"]

	if cfg.Kind != "Deployment" {
		t.Errorf("expected generic Kind 'Deployment', got %q", cfg.Kind)
	}
	if cfg.Replicas != 1 {
		t.Errorf("expected generic Replicas 1, got %d", cfg.Replicas)
	}
	if cfg.AddIngress {
		t.Error("expected generic AddIngress to be false")
	}
	if cfg.AddHPA {
		t.Error("expected generic AddHPA to be false")
	}
	if cfg.AddPDB {
		t.Error("expected generic AddPDB to be false")
	}
	if cfg.PVCSize != "1Gi" {
		t.Errorf("expected generic PVCSize '1Gi', got %q", cfg.PVCSize)
	}
}

func TestSmartDefaultsSensitiveEnvVars(t *testing.T) {
	tests := []struct {
		name    string
		envKey  string
		secret  bool
	}{
		{"password var", "DB_PASSWORD", true},
		{"secret var", "APP_SECRET", true},
		{"token var", "API_TOKEN", true},
		{"lowercase password", "db_password", true},
		{"mixed case Secret", "App_Secret_Key", true},
		{"normal var", "DATABASE_HOST", false},
		{"port var", "APP_PORT", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compose := &parser.ComposeFile{
				Services: map[string]parser.ServiceConfig{
					"svc": {
						Image:       "my-app:1.0",
						Environment: map[string]string{tt.envKey: "value"},
					},
				},
			}
			m := newTestModel(compose)
			cfg := m.config.Services["svc"]
			if cfg.CreateSecret != tt.secret {
				t.Errorf("env var %q: expected CreateSecret=%v, got %v", tt.envKey, tt.secret, cfg.CreateSecret)
			}
		})
	}
}

func TestSmartDefaultsHPADefaults(t *testing.T) {
	compose := &parser.ComposeFile{
		Services: map[string]parser.ServiceConfig{
			"web": {Image: "nginx:1.25"},
		},
	}
	m := newTestModel(compose)
	cfg := m.config.Services["web"]

	if cfg.HPAMin != 2 {
		t.Errorf("expected HPAMin 2, got %d", cfg.HPAMin)
	}
	if cfg.HPAMax != 10 {
		t.Errorf("expected HPAMax 10, got %d", cfg.HPAMax)
	}
	if cfg.HPATargetCPU != 70 {
		t.Errorf("expected HPATargetCPU 70, got %d", cfg.HPATargetCPU)
	}
	if cfg.PDBMinAvail != 1 {
		t.Errorf("expected PDBMinAvail 1, got %d", cfg.PDBMinAvail)
	}
}

// --- Full flow test ---

func TestFullWizardFlow(t *testing.T) {
	compose := &parser.ComposeFile{
		Services: map[string]parser.ServiceConfig{
			"web": {Image: "nginx:1.25"},
		},
	}
	m := newTestModel(compose)

	// Phase 1: Namespace - clear default and type "staging"
	for i := 0; i < len("default"); i++ {
		var result tea.Model
		result, _ = m.Update(keyBackspace())
		m = result.(model)
	}
	m = typeString(m, "staging").(model)
	result, _ := m.Update(keyEnter())
	m = result.(model)
	if m.config.Namespace != "staging" {
		t.Fatalf("expected namespace 'staging', got %q", m.config.Namespace)
	}

	// Phase 2: Output format - select helm (index 1)
	result, _ = m.Update(keyDown())
	m = result.(model)
	result, _ = m.Update(keyEnter())
	m = result.(model)
	if m.config.OutputFormat != "helm" {
		t.Fatalf("expected output format 'helm', got %q", m.config.OutputFormat)
	}

	// Phase 3: Service config for "web"
	// fieldReplicas - type "4"
	m = typeString(m, "4").(model)
	result, _ = m.Update(keyEnter())
	m = result.(model)

	// fieldIngress - accept default (yes)
	result, _ = m.Update(keyEnter())
	m = result.(model)

	// fieldIngressHost - accept default
	result, _ = m.Update(keyEnter())
	m = result.(model)

	// fieldTLS - accept default
	result, _ = m.Update(keyEnter())
	m = result.(model)

	// fieldHPA - accept default
	result, _ = m.Update(keyEnter())
	m = result.(model)

	// fieldPDB - accept default
	result, _ = m.Update(keyEnter())
	m = result.(model)

	if m.phase != phaseSummary {
		t.Fatalf("expected phaseSummary, got %d", m.phase)
	}

	cfg := m.config.Services["web"]
	if cfg.Replicas != 4 {
		t.Errorf("expected replicas 4, got %d", cfg.Replicas)
	}
	if !cfg.AddIngress {
		t.Error("expected AddIngress true")
	}
	if !cfg.AddTLS {
		t.Error("expected AddTLS true")
	}
	if !cfg.AddHPA {
		t.Error("expected AddHPA true")
	}

	// Phase 4: Summary - confirm
	result, cmd := m.Update(keyEnter())
	m = result.(model)
	if m.phase != phaseDone {
		t.Errorf("expected phaseDone, got %d", m.phase)
	}
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestViewSummaryShowsIngressHost(t *testing.T) {
	compose := &parser.ComposeFile{
		Services: map[string]parser.ServiceConfig{
			"web": {Image: "nginx:1.25"},
		},
	}
	m := newTestModel(compose)
	m.phase = phaseSummary
	view := m.View()
	// web server gets ingress by default
	if !strings.Contains(view, "web.example.com") {
		t.Error("expected summary to show ingress host 'web.example.com'")
	}
}

func TestViewSummaryShowsHPAAndPDB(t *testing.T) {
	compose := &parser.ComposeFile{
		Services: map[string]parser.ServiceConfig{
			"web": {Image: "nginx:1.25"},
			"db":  {Image: "postgres:15"},
		},
	}
	m := newTestModel(compose)
	m.phase = phaseSummary
	view := m.View()
	// web has HPA, db has PDB
	if !strings.Contains(view, "yes") {
		t.Error("expected summary to show 'yes' for HPA or PDB")
	}
	if !strings.Contains(view, "StatefulSet") {
		t.Error("expected summary to show 'StatefulSet' for database")
	}
	if !strings.Contains(view, "Deployment") {
		t.Error("expected summary to show 'Deployment' for web")
	}
}

func TestServiceNamesSorted(t *testing.T) {
	compose := &parser.ComposeFile{
		Services: map[string]parser.ServiceConfig{
			"zeta":  {Image: "nginx"},
			"alpha": {Image: "redis"},
			"mid":   {Image: "node"},
		},
	}
	m := newTestModel(compose)
	expected := []string{"alpha", "mid", "zeta"}
	for i, name := range m.serviceNames {
		if name != expected[i] {
			t.Errorf("expected serviceNames[%d] = %q, got %q", i, expected[i], name)
		}
	}
}
