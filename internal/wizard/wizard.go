package wizard

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

// Run launches the interactive wizard TUI.
// TODO: implement in Prompt 5
func Run() (*WizardConfig, error) {
	return nil, nil
}
