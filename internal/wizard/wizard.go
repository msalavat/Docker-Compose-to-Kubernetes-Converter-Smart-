package wizard

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/compositor/kompoze/internal/parser"
	tea "github.com/charmbracelet/bubbletea"
)

// WizardConfig holds the result of the interactive wizard session.
type WizardConfig struct {
	Namespace    string
	OutputFormat string // "manifests" | "helm" | "kustomize"
	Services     map[string]ServiceWizardConfig
}

// ServiceWizardConfig holds per-service wizard choices.
type ServiceWizardConfig struct {
	Kind         string // "Deployment" | "StatefulSet"
	Replicas     int32
	AddIngress   bool
	IngressHost  string
	AddTLS       bool
	AddHPA       bool
	HPAMin       int32
	HPAMax       int32
	HPATargetCPU int32
	AddPDB       bool
	PDBMinAvail  int32
	PVCSize      string
	CreateSecret bool
}

// phase tracks the wizard's current step.
type phase int

const (
	phaseNamespace phase = iota
	phaseOutputFormat
	phaseServiceConfig
	phaseSummary
	phaseDone
)

// fieldType tracks which field we're editing for a service.
type fieldType int

const (
	fieldReplicas fieldType = iota
	fieldIngress
	fieldIngressHost
	fieldTLS
	fieldHPA
	fieldPDB
	fieldDone
)

type model struct {
	compose      *parser.ComposeFile
	config       WizardConfig
	serviceNames []string
	serviceIdx   int
	phase        phase
	field        fieldType
	input        string
	cursor       int // for list selection
	quitting     bool
	err          error
}

// Run launches the interactive wizard TUI.
func Run(compose *parser.ComposeFile) (*WizardConfig, error) {
	if compose == nil {
		return nil, fmt.Errorf("compose file is nil")
	}

	names := make([]string, 0, len(compose.Services))
	for name := range compose.Services {
		names = append(names, name)
	}
	sort.Strings(names)

	// Build initial service configs with smart defaults
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

		// Check for sensitive env vars
		for k := range svc.Environment {
			upper := strings.ToUpper(k)
			if strings.Contains(upper, "PASSWORD") || strings.Contains(upper, "SECRET") || strings.Contains(upper, "TOKEN") {
				cfg.CreateSecret = true
				break
			}
		}

		services[name] = cfg
	}

	m := model{
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

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("wizard error: %w", err)
	}

	fm := finalModel.(model)
	if fm.err != nil {
		return nil, fm.err
	}
	if fm.quitting {
		return nil, fmt.Errorf("wizard cancelled")
	}

	return &fm.config, nil
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.phase != phaseNamespace && m.phase != phaseServiceConfig {
				m.quitting = true
				return m, tea.Quit
			}
			if msg.String() == "ctrl+c" {
				m.quitting = true
				return m, tea.Quit
			}
		}

		switch m.phase {
		case phaseNamespace:
			return m.updateNamespace(msg)
		case phaseOutputFormat:
			return m.updateOutputFormat(msg)
		case phaseServiceConfig:
			return m.updateServiceConfig(msg)
		case phaseSummary:
			return m.updateSummary(msg)
		}
	}
	return m, nil
}

func (m model) updateNamespace(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if m.input != "" {
			m.config.Namespace = m.input
		}
		m.phase = phaseOutputFormat
		m.cursor = 0
		m.input = ""
	case "backspace":
		if len(m.input) > 0 {
			m.input = m.input[:len(m.input)-1]
		}
	default:
		if len(msg.String()) == 1 {
			m.input += msg.String()
		}
	}
	return m, nil
}

func (m model) updateOutputFormat(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < 2 {
			m.cursor++
		}
	case "enter":
		formats := []string{"manifests", "helm", "kustomize"}
		m.config.OutputFormat = formats[m.cursor]
		m.phase = phaseServiceConfig
		m.serviceIdx = 0
		m.field = fieldReplicas
		m.input = ""
	}
	return m, nil
}

func (m model) updateServiceConfig(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	name := m.serviceNames[m.serviceIdx]
	cfg := m.config.Services[name]

	switch msg.String() {
	case "enter":
		switch m.field {
		case fieldReplicas:
			if m.input != "" {
				if v, err := strconv.Atoi(m.input); err == nil {
					cfg.Replicas = int32(v)
				}
			}
			m.field = fieldIngress
			m.input = ""
		case fieldIngress:
			cfg.AddIngress = m.input != "n" && m.input != "N"
			m.field = fieldHPA
			m.input = ""
			if cfg.AddIngress {
				m.field = fieldIngressHost
			}
		case fieldIngressHost:
			if m.input != "" {
				cfg.IngressHost = m.input
			}
			m.field = fieldTLS
			m.input = ""
		case fieldTLS:
			cfg.AddTLS = m.input != "n" && m.input != "N"
			m.field = fieldHPA
			m.input = ""
		case fieldHPA:
			cfg.AddHPA = m.input != "n" && m.input != "N"
			m.field = fieldPDB
			m.input = ""
		case fieldPDB:
			cfg.AddPDB = m.input != "n" && m.input != "N"
			m.config.Services[name] = cfg
			// Move to next service or summary
			m.serviceIdx++
			if m.serviceIdx >= len(m.serviceNames) {
				m.phase = phaseSummary
			} else {
				m.field = fieldReplicas
			}
			m.input = ""
			return m, nil
		}
		m.config.Services[name] = cfg
	case "backspace":
		if len(m.input) > 0 {
			m.input = m.input[:len(m.input)-1]
		}
	default:
		if len(msg.String()) == 1 {
			m.input += msg.String()
		}
	}
	return m, nil
}

func (m model) updateSummary(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", "y", "Y":
		m.phase = phaseDone
		return m, tea.Quit
	case "n", "N":
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}

func (m model) View() string {
	var s strings.Builder

	s.WriteString(titleStyle.Render("Kompoze Wizard"))
	s.WriteString("\n")
	s.WriteString(fmt.Sprintf("Found %d services: %s\n\n", len(m.serviceNames), strings.Join(m.serviceNames, ", ")))

	switch m.phase {
	case phaseNamespace:
		s.WriteString(questionStyle.Render("? Namespace: "))
		s.WriteString(fmt.Sprintf("[%s] %s", m.config.Namespace, m.input))
		s.WriteString("\n")

	case phaseOutputFormat:
		s.WriteString(questionStyle.Render("? Output format:"))
		s.WriteString("\n")
		formats := []string{"Kubernetes manifests", "Helm chart", "Kustomize (base + overlays)"}
		for i, f := range formats {
			if i == m.cursor {
				s.WriteString(selectedStyle.Render("  > " + f))
			} else {
				s.WriteString(dimStyle.Render("    " + f))
			}
			s.WriteString("\n")
		}

	case phaseServiceConfig:
		name := m.serviceNames[m.serviceIdx]
		svc := m.compose.Services[name]
		st := DetectServiceType(svc.Image)
		cfg := m.config.Services[name]

		s.WriteString(serviceHeaderStyle.Render(fmt.Sprintf("Service: %s (%s) [%s]", name, svc.Image, st)))
		s.WriteString("\n\n")

		switch m.field {
		case fieldReplicas:
			s.WriteString(questionStyle.Render(fmt.Sprintf("? Replicas: [%d] ", cfg.Replicas)))
			s.WriteString(m.input)
		case fieldIngress:
			def := "Y/n"
			if !cfg.AddIngress {
				def = "y/N"
			}
			s.WriteString(questionStyle.Render(fmt.Sprintf("? Add Ingress? [%s] ", def)))
			s.WriteString(m.input)
		case fieldIngressHost:
			s.WriteString(questionStyle.Render(fmt.Sprintf("? Ingress hostname: [%s] ", cfg.IngressHost)))
			s.WriteString(m.input)
		case fieldTLS:
			s.WriteString(questionStyle.Render("? Add TLS (cert-manager)? [Y/n] "))
			s.WriteString(m.input)
		case fieldHPA:
			def := "Y/n"
			if !cfg.AddHPA {
				def = "y/N"
			}
			s.WriteString(questionStyle.Render(fmt.Sprintf("? Add HPA (auto-scaling)? [%s] ", def)))
			s.WriteString(m.input)
		case fieldPDB:
			def := "Y/n"
			if !cfg.AddPDB {
				def = "y/N"
			}
			s.WriteString(questionStyle.Render(fmt.Sprintf("? Add PodDisruptionBudget? [%s] ", def)))
			s.WriteString(m.input)
		}
		s.WriteString("\n")

	case phaseSummary:
		s.WriteString(successStyle.Render("Summary:"))
		s.WriteString("\n\n")
		s.WriteString(fmt.Sprintf("  Namespace: %s\n", m.config.Namespace))
		s.WriteString(fmt.Sprintf("  Format:    %s\n\n", m.config.OutputFormat))

		header := fmt.Sprintf("  %-15s %-12s %-8s %-8s %-5s %-5s", "Service", "Kind", "Replicas", "Ingress", "HPA", "PDB")
		s.WriteString(dimStyle.Render(header))
		s.WriteString("\n")
		s.WriteString(dimStyle.Render("  " + strings.Repeat("-", 55)))
		s.WriteString("\n")

		for _, name := range m.serviceNames {
			cfg := m.config.Services[name]
			ingress := "-"
			if cfg.AddIngress {
				ingress = cfg.IngressHost
			}
			hpa := "-"
			if cfg.AddHPA {
				hpa = "yes"
			}
			pdb := "-"
			if cfg.AddPDB {
				pdb = "yes"
			}
			line := fmt.Sprintf("  %-15s %-12s %-8d %-8s %-5s %-5s", name, cfg.Kind, cfg.Replicas, ingress, hpa, pdb)
			s.WriteString(line + "\n")
		}

		s.WriteString("\n")
		s.WriteString(questionStyle.Render("? Generate manifests? [Y/n] "))

	case phaseDone:
		s.WriteString(successStyle.Render("Generating manifests..."))
		s.WriteString("\n")
	}

	return s.String()
}
