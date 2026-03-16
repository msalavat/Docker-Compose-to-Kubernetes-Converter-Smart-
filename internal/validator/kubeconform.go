package validator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"

	"github.com/compositor/kompoze/internal/converter"
	sigsyaml "sigs.k8s.io/yaml"
)

// kubeconformResult represents a single result from kubeconform JSON output.
type kubeconformResult struct {
	Filename string `json:"filename"`
	Kind     string `json:"kind"`
	Name     string `json:"name"`
	Version  string `json:"version"`
	Status   string `json:"status"`
	Msg      string `json:"msg"`
}

// ValidateWithKubeconform runs kubeconform schema validation on the generated manifests.
// Returns nil if kubeconform is not installed.
func ValidateWithKubeconform(result *converter.ConvertResult) []ValidationError {
	return validateWithKubeconformInternal(result, exec.LookPath)
}

// validateWithKubeconformInternal is the internal implementation that accepts a lookPath function for testing.
func validateWithKubeconformInternal(result *converter.ConvertResult, lookPath func(string) (string, error)) []ValidationError {
	binary := "kubeconform"
	if runtime.GOOS == "windows" {
		binary = "kubeconform.exe"
	}

	_, err := lookPath(binary)
	if err != nil {
		return []ValidationError{
			{
				Resource: "",
				Field:    "",
				Severity: "info",
				Message:  "kubeconform not found, skipping schema validation",
			},
		}
	}

	// Collect all resources as runtime.Object-like items to serialize
	var manifests []interface{}

	for i := range result.Deployments {
		manifests = append(manifests, &result.Deployments[i])
	}
	for i := range result.StatefulSets {
		manifests = append(manifests, &result.StatefulSets[i])
	}
	for i := range result.Services {
		manifests = append(manifests, &result.Services[i])
	}
	for i := range result.ConfigMaps {
		manifests = append(manifests, &result.ConfigMaps[i])
	}
	for i := range result.Secrets {
		manifests = append(manifests, &result.Secrets[i])
	}
	for i := range result.PVCs {
		manifests = append(manifests, &result.PVCs[i])
	}
	for i := range result.Ingresses {
		manifests = append(manifests, &result.Ingresses[i])
	}
	for i := range result.HPAs {
		manifests = append(manifests, &result.HPAs[i])
	}
	for i := range result.PDBs {
		manifests = append(manifests, &result.PDBs[i])
	}
	for i := range result.ServiceAccounts {
		manifests = append(manifests, &result.ServiceAccounts[i])
	}
	for i := range result.NetworkPolicies {
		manifests = append(manifests, &result.NetworkPolicies[i])
	}

	if len(manifests) == 0 {
		return nil
	}

	// Build a multi-document YAML to pipe to kubeconform
	var yamlBuf bytes.Buffer
	for _, m := range manifests {
		data, err := sigsyaml.Marshal(m)
		if err != nil {
			return []ValidationError{
				{
					Resource: "",
					Field:    "",
					Severity: "error",
					Message:  fmt.Sprintf("failed to marshal manifest to YAML: %v", err),
				},
			}
		}
		yamlBuf.WriteString("---\n")
		yamlBuf.Write(data)
	}

	// Run kubeconform
	cmd := exec.Command(binary, "-summary", "-output", "json")
	cmd.Stdin = &yamlBuf
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// kubeconform exits non-zero on validation failures, so we don't treat that as fatal
	_ = cmd.Run()

	return parseKubeconformOutput(stdout.Bytes())
}

// parseKubeconformOutput parses kubeconform JSON output lines into ValidationErrors.
func parseKubeconformOutput(data []byte) []ValidationError {
	var errors []ValidationError

	// kubeconform outputs one JSON object per line
	decoder := json.NewDecoder(bytes.NewReader(data))
	for decoder.More() {
		var res kubeconformResult
		if err := decoder.Decode(&res); err != nil {
			// Skip unparseable lines (e.g., summary lines)
			continue
		}

		resource := fmt.Sprintf("%s/%s", res.Kind, res.Name)

		switch res.Status {
		case "statusInvalid":
			errors = append(errors, ValidationError{
				Resource: resource,
				Field:    "",
				Severity: "error",
				Message:  fmt.Sprintf("kubeconform: %s", res.Msg),
			})
		case "statusError":
			errors = append(errors, ValidationError{
				Resource: resource,
				Field:    "",
				Severity: "error",
				Message:  fmt.Sprintf("kubeconform: %s", res.Msg),
			})
		case "statusValid":
			// Valid, nothing to report
		}
	}

	return errors
}
