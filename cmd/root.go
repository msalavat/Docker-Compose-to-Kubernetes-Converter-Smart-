// Package cmd implements the CLI commands for kompoze.
package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "kompoze",
	Short: "Convert docker-compose.yml to production-ready Kubernetes manifests",
	Long: `kompoze is a smart Docker Compose to Kubernetes converter.

Unlike basic converters, kompoze generates production-grade manifests with:
  - Resource limits and requests (auto-detected or smart defaults)
  - Health probes (liveness + readiness)
  - Security context (restrictive by default)
  - Helm chart and Kustomize output
  - Interactive wizard mode

Usage:
  kompoze convert docker-compose.yml -o k8s/
  kompoze convert --wizard docker-compose.yml
  kompoze convert --helm -o helm-chart/
  kompoze convert --kustomize -o kustomize/`,
}

var appVersion = "dev"

// SetVersion sets the application version (called from main).
func SetVersion(v string) {
	appVersion = v
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
