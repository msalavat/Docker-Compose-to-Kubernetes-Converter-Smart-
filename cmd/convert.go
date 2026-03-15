package cmd

import (
	"fmt"
	"os"

	"github.com/compositor/kompoze/internal/converter"
	"github.com/compositor/kompoze/internal/output"
	"github.com/compositor/kompoze/internal/parser"
	"github.com/spf13/cobra"
)

var (
	outputDir    string
	namespace    string
	appName      string
	helmOutput   bool
	kustomizeOut bool
	wizardMode   bool
	validateFlag bool
	strictFlag   bool
	noProbes     bool
	noResources  bool
	noSecurity   bool
	noNetPolicy  bool
	singleFile   bool
	quietFlag    bool
	verboseFlag  bool
	dryRun       bool
)

func init() {
	rootCmd.AddCommand(convertCmd)

	convertCmd.Flags().StringVarP(&outputDir, "output", "o", "./k8s", "Output directory")
	convertCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Kubernetes namespace")
	convertCmd.Flags().StringVar(&appName, "app-name", "", "Application name (default: from compose file name)")
	convertCmd.Flags().BoolVar(&helmOutput, "helm", false, "Generate Helm chart")
	convertCmd.Flags().BoolVar(&kustomizeOut, "kustomize", false, "Generate Kustomize structure")
	convertCmd.Flags().BoolVar(&wizardMode, "wizard", false, "Interactive wizard mode")
	convertCmd.Flags().BoolVar(&validateFlag, "validate", false, "Validate generated manifests")
	convertCmd.Flags().BoolVar(&strictFlag, "strict", false, "Fail on validation warnings")
	convertCmd.Flags().BoolVar(&noProbes, "no-probes", false, "Skip default health probes")
	convertCmd.Flags().BoolVar(&noResources, "no-resources", false, "Skip default resource limits")
	convertCmd.Flags().BoolVar(&noSecurity, "no-security", false, "Skip default security context")
	convertCmd.Flags().BoolVar(&noNetPolicy, "no-network-policy", false, "Skip NetworkPolicy generation")
	convertCmd.Flags().BoolVar(&singleFile, "single-file", false, "Output all manifests in single file")
	convertCmd.Flags().BoolVarP(&quietFlag, "quiet", "q", false, "Suppress non-error output")
	convertCmd.Flags().BoolVarP(&verboseFlag, "verbose", "v", false, "Verbose output")
	convertCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print manifests to stdout, don't write files")
}

var convertCmd = &cobra.Command{
	Use:   "convert [docker-compose.yml]",
	Short: "Convert a docker-compose file to Kubernetes manifests",
	Long: `Convert a docker-compose.yml file to production-ready Kubernetes manifests.

Examples:
  kompoze convert docker-compose.yml -o k8s/
  kompoze convert --wizard docker-compose.yml
  kompoze convert --helm -o helm-chart/
  kompoze convert --kustomize -o kustomize/
  kompoze convert --dry-run docker-compose.yml`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		composeFile := "docker-compose.yml"
		if len(args) > 0 {
			composeFile = args[0]
		}

		// Parse
		if !quietFlag {
			fmt.Printf("Parsing %s...", composeFile)
		}
		compose, err := parser.ParseComposeFile(composeFile)
		if err != nil {
			return fmt.Errorf("parsing compose file: %w", err)
		}
		if !quietFlag {
			fmt.Printf(" ✓ (%d services found)\n", len(compose.Services))
		}

		// Convert
		opts := converter.ConvertOptions{
			OutputDir:    outputDir,
			Namespace:    namespace,
			AppName:      appName,
			AddProbes:    !noProbes,
			AddResources: !noResources,
			AddSecurity:  !noSecurity,
			SingleFile:   singleFile,
		}

		if !quietFlag {
			fmt.Println("Generating manifests...")
		}
		result, err := converter.Convert(compose, opts)
		if err != nil {
			return fmt.Errorf("converting: %w", err)
		}

		// Output
		if dryRun {
			content, err := output.RenderManifests(result)
			if err != nil {
				return fmt.Errorf("rendering manifests: %w", err)
			}
			fmt.Fprint(os.Stdout, content)
			return nil
		}

		if err := output.WriteManifests(result, outputDir, singleFile); err != nil {
			return fmt.Errorf("writing manifests: %w", err)
		}

		if !quietFlag {
			total := len(result.Deployments) + len(result.Services) + len(result.ConfigMaps) + len(result.PVCs)
			fmt.Printf("\nOutput written to %s/ (%d files)\n", outputDir, total)
		}

		return nil
	},
}
