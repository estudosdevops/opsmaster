package install

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/spf13/cobra"

	"github.com/estudosdevops/opsmaster/internal/cloud"
	"github.com/estudosdevops/opsmaster/internal/cloud/provider"
	"github.com/estudosdevops/opsmaster/internal/csv"
	"github.com/estudosdevops/opsmaster/internal/executor"
	"github.com/estudosdevops/opsmaster/internal/installer"
	"github.com/estudosdevops/opsmaster/internal/logger"
)

// Puppet command flags
var (
	instancesFile   string // CSV file with instance list
	puppetServer    string // Puppet Server hostname
	puppetPort      int    // Puppet Server port
	puppetVersion   string // Puppet version to install
	environment     string // Puppet environment
	customFactsFile string // YAML file with custom facts definitions
	maxConcurrency  int    // Max parallel executions
	awsProfile      string // AWS profile to use
	dryRun          bool   // Simulate without executing
	skipValidation  bool   // Skip prerequisite validation
)

// totalSteps is the total number of steps in the Puppet installation process.
const totalSteps = 6

// logStep logs a numbered step in the installation process.
// This ensures consistent formatting across all steps.
func logStep(log *slog.Logger, step int, description string) {
	log.Info(fmt.Sprintf("📋 Step %d/%d: %s", step, totalSteps, description))
}

// fatalError logs an error and returns exit code 1.
// Use for unrecoverable errors during initialization.
func fatalError(log *slog.Logger, message string, err error) error {
	log.Error(message, "error", err)
	return fmt.Errorf("%s: %w", message, err)
}

// puppetCmd represents the puppet installation command
var puppetCmd = &cobra.Command{
	Use:   "puppet",
	Short: "Instala Puppet Agent em instâncias na nuvem",
	Long: `Instala e configura Puppet Agent em múltiplas instâncias na nuvem em paralelo.

Lê lista de instâncias de arquivo CSV, valida pré-requisitos, instala Puppet Agent,
configura puppet.conf com certname único e cria tags nas instâncias após instalação bem-sucedida.

Formato CSV:
  instance_id,account,region,cloud,environment
  i-0123456789abcdef0,111111111111,us-east-1,aws,production
  i-fedcba9876543210,111111111111,us-west-2,aws,staging

O arquivo CSV deve ter cabeçalhos: instance_id, account, region
Colunas opcionais: cloud (padrão aws), environment, quaisquer colunas extras são armazenadas como metadados.

Custom Facter Facts:
  Por padrão, o OpsMaster cria automaticamente um arquivo location.yaml em
  /opt/puppetlabs/facter/facts.d/ com os seguintes campos do CSV:
    - account
    - environment
    - region

  Para customizar os facts criados, use --custom-facts com arquivo YAML:

  Exemplo custom-facts.yaml:
    location:
      file_path: "location.yaml"
      fact_name: "location"
      fields:
        account: "account"
        environment: "environment"
        region: "region"
    compliance:
      file_path: "compliance.yaml"
      fact_name: "compliance"
      fields:
        compliance_level: "compliance"
        data_classification: "classification"

Exemplos:
  # Instalação básica (cria location.yaml automaticamente)
  opsmaster install puppet \
    --instances-file instances.csv \
    --puppet-server puppet.example.com

  # Com custom facts personalizados
  opsmaster install puppet \
    --instances-file instances.csv \
    --puppet-server puppet.example.com \
    --custom-facts custom-facts.yaml

  # Com configurações customizadas
  opsmaster install puppet \
    --instances-file instances.csv \
    --puppet-server puppet.example.com \
    --puppet-port 8140 \
    --puppet-version 7 \
    --environment production \
    --max-concurrency 20

  # Dry run (simular)
  opsmaster install puppet \
    --instances-file instances.csv \
    --puppet-server puppet.example.com \
    --dry-run`,

	RunE: runPuppetInstall,
}

func init() {
	// Register puppet subcommand
	InstallCmd.AddCommand(puppetCmd)

	// Required flags
	puppetCmd.Flags().StringVar(&instancesFile, "instances-file", "", "Arquivo CSV com lista de instâncias (obrigatório)")
	puppetCmd.Flags().StringVar(&puppetServer, "puppet-server", "", "Hostname do Puppet Server (obrigatório)")
	puppetCmd.MarkFlagRequired("instances-file")
	puppetCmd.MarkFlagRequired("puppet-server")

	// Optional flags with defaults
	puppetCmd.Flags().IntVar(&puppetPort, "puppet-port", 8140, "Porta do Puppet Server")
	puppetCmd.Flags().StringVar(&puppetVersion, "puppet-version", "7", "Versão do Puppet a instalar")
	puppetCmd.Flags().StringVar(&environment, "environment", "production", "Ambiente Puppet")
	puppetCmd.Flags().StringVar(&customFactsFile, "custom-facts", "", "Arquivo YAML com definições de custom facts (opcional)")
	puppetCmd.Flags().IntVar(&maxConcurrency, "max-concurrency", 10, "Máximo de instalações paralelas")
	puppetCmd.Flags().StringVar(&awsProfile, "aws-profile", "", "Perfil AWS a usar (padrão: perfil default)")
	puppetCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Simular instalação sem executar")
	puppetCmd.Flags().BoolVar(&skipValidation, "skip-validation", false, "Pular validação de pré-requisitos (não recomendado)")
}

// runPuppetInstall orchestrates the entire Puppet installation workflow
func runPuppetInstall(cmd *cobra.Command, args []string) error {
	// Create logger
	log := logger.Get()

	startTime := time.Now()
	log.Info("🚀 Puppet Installation Started",
		"instances_file", instancesFile,
		"puppet_server", puppetServer,
		"max_concurrency", maxConcurrency,
		"dry_run", dryRun,
	)

	// Create context with cancellation support (Ctrl+C)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ============================================================
	// STEP 1: Parse CSV file and load instances
	// ============================================================
	logStep(log, 1, "Parsing CSV file")
	log.Info("📄 Reading instances", "file", instancesFile)

	instances, err := parseInstancesFile(instancesFile)
	if err != nil {
		return fatalError(log, "Failed to parse CSV file", err)
	}

	log.Info("✅ CSV parsed successfully", "total_instances", len(instances))
	if len(instances) == 0 {
		return fmt.Errorf("no instances found in CSV file")
	}

	// ============================================================
	// STEP 2: Initialize cloud provider
	// ============================================================
	logStep(log, 2, "Initializing cloud provider")

	// Detect cloud provider from instances
	cloudType, err := provider.DetectCloudFromInstances(instances)
	if err != nil {
		return fatalError(log, "Failed to detect cloud provider", err)
	}

	log.Info("☁️  Detected cloud provider", "cloud", cloudType)

	// Create provider using factory
	var providerOptions []provider.Option
	if awsProfile != "" {
		providerOptions = append(providerOptions, provider.WithProfile(awsProfile))
		log.Info("   Using AWS profile", "profile", awsProfile)
	}

	cloudProvider, err := provider.NewProvider(cloudType, providerOptions...)
	if err != nil {
		return fatalError(log, "Failed to create cloud provider", err)
	}

	log.Info("✅ Cloud provider initialized", "provider", cloudProvider.Name())

	// ============================================================
	// STEP 3: Load custom facts configuration
	// ============================================================
	logStep(log, 3, "Loading custom facts configuration")

	var customFacts map[string]installer.FactDefinition

	if customFactsFile != "" {
		// Load custom facts from YAML file
		customFacts, err = installer.LoadCustomFactsFromYAML(customFactsFile)
		if err != nil {
			return fatalError(log, "Failed to load custom facts", err)
		}

		// Log loaded facts details
		log.Info("✅ Custom facts loaded from file",
			"file", customFactsFile,
			"fact_count", len(customFacts),
		)

		// Log each fact file that will be created
		for _, factDef := range customFacts {
			log.Info("   → Fact file configured",
				"file", factDef.FilePath,
				"fact_name", factDef.FactName,
				"field_count", len(factDef.Fields),
			)
		}
	} else {
		// Use default custom facts (location.yaml)
		customFacts = installer.GetDefaultCustomFacts()
		log.Info("✅ Using default custom facts (no --custom-facts flag provided)")
		log.Info("   → location.yaml will be created with: account, environment, region")
	}

	// Validate that CSV columns required by facts exist
	if len(instances) > 0 {
		installer.LogMissingFactColumns(log, customFacts, instances[0])
	}

	// ============================================================
	// STEP 4: Create Puppet installer
	// ============================================================
	logStep(log, 4, "Creating Puppet installer")

	puppetInstaller := installer.NewPuppetInstaller(installer.PuppetOptions{
		Server:      puppetServer,
		Port:        puppetPort,
		Version:     puppetVersion,
		Environment: environment,
		CustomFacts: customFacts,
	})

	log.Info("✅ Puppet installer created",
		"server", puppetServer,
		"port", puppetPort,
		"version", puppetVersion,
		"environment", environment,
		"custom_facts_enabled", len(customFacts) > 0,
	)

	// ============================================================
	// STEP 5: Setup skip validation flag
	// ============================================================
	logStep(log, 5, "Configuring validation settings")
	if skipValidation {
		log.Warn("⚠️  Validation skipped (--skip-validation enabled)")
	} else {
		log.Info("🔍 Validation will be performed (SSM + Puppet Server connectivity)")
	}

	// ============================================================
	// STEP 6: Execute parallel installation
	// ============================================================
	logStep(log, 6, "Starting parallel installation")
	log.Info("⚡ Executing installation",
		"total_instances", len(instances),
		"max_concurrency", maxConcurrency,
		"dry_run", dryRun,
	)

	if dryRun {
		log.Warn("🔍 DRY RUN MODE: No changes will be made")
	}

	// Create parallel executor
	exec := executor.NewParallelExecutor(executor.ExecutorConfig{
		Provider:       cloudProvider,
		Installer:      puppetInstaller,
		MaxConcurrency: maxConcurrency,
		SkipValidation: skipValidation,
		SkipTagging:    false,
		DryRun:         dryRun,
	})

	// Execute installation on all instances
	result, err := exec.Execute(ctx, instances)
	if err != nil {
		log.Error("Failed to execute installation", "error", err)
		return fmt.Errorf("execution failed: %w", err)
	}

	// ============================================================
	// STEP 6: Report results
	// ============================================================
	duration := time.Since(startTime)

	log.Info("📊 Installation Summary",
		"total", result.Total,
		"successful", result.Success,
		"failed", result.Failed,
		"skipped", result.Skipped,
		"duration", duration.Round(time.Second).String(),
	)

	// Print detailed results
	printResults(result)

	// Exit with error if any installations failed
	if result.Failed > 0 {
		return fmt.Errorf("installation failed for %d instances", result.Failed)
	}

	log.Info("✅ All installations completed successfully!")
	return nil
}

// parseInstancesFile parses CSV file and returns list of instances
func parseInstancesFile(filePath string) ([]*cloud.Instance, error) {
	// Create CSV parser with configuration
	parser := csv.NewParser(csv.CSVConfig{
		HasHeader:      true, // Expect header row
		RequiredFields: []string{"instance_id", "account", "region"},
		CloudDefault:   "aws",
		Delimiter:      ',',
	})

	// Parse file (ParseFile expects filePath string, not *os.File)
	instances, err := parser.ParseFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	return instances, nil
}

// printInstanceMetadata prints metadata information for an instance.
// This centralizes metadata display logic to avoid duplication between
// successful and failed installation reports.
//
// Metadata includes:
//   - OS type (debian/rhel)
//   - Certname used for Puppet
//   - Whether certname was preserved from previous installation
func printInstanceMetadata(metadata map[string]string) {
	if metadata == nil {
		return
	}

	// Print OS type if available
	if os := metadata["os"]; os != "" {
		fmt.Printf("    OS: %s\n", os)
	}

	// Print certname if available
	if certname := metadata["certname"]; certname != "" {
		fmt.Printf("    Certname: %s\n", certname)

		// Indicate if certname was preserved from previous installation
		if preserved := metadata["certname_preserved"]; preserved == "true" {
			fmt.Printf("    (Certname preserved from previous installation)\n")
		}
	}
}

// printResults prints detailed results to console
func printResults(result *executor.AggregatedResult) {
	// Print successful installations
	if len(result.Results) > 0 {
		fmt.Println("\n" + "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("📋 Detailed Results")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		// Group by status
		successful := []*executor.ExecutionResult{}
		failed := []*executor.ExecutionResult{}
		skipped := []*executor.ExecutionResult{}

		for _, r := range result.Results {
			switch r.Status {
			case executor.StatusSuccess:
				successful = append(successful, r)
			case executor.StatusFailed:
				failed = append(failed, r)
			case executor.StatusSkipped:
				skipped = append(skipped, r)
			}
		}

		// Print successful
		if len(successful) > 0 {
			fmt.Println("\n✅ Successful Installations:")
			for _, r := range successful {
				fmt.Printf("  • %s (%s/%s)\n", r.Instance.ID, r.Instance.Account, r.Instance.Region)

				// Print metadata (OS, certname, etc)
				printInstanceMetadata(r.Metadata)

				// Print duration
				duration := r.Duration.Round(time.Second)
				fmt.Printf("    Duration: %s\n", duration)
			}
		}

		// Print failed
		if len(failed) > 0 {
			fmt.Println("\n❌ Failed Installations:")
			for _, r := range failed {
				fmt.Printf("  • %s (%s/%s)\n", r.Instance.ID, r.Instance.Account, r.Instance.Region)

				// Print metadata (shows detected OS even on failure)
				printInstanceMetadata(r.Metadata)

				if err := r.GetError(); err != nil {
					fmt.Printf("    Error: %s\n", err)
				}
			}
		}

		// Print skipped
		if len(skipped) > 0 {
			fmt.Println("\n⏭️  Skipped Installations:")
			for _, r := range skipped {
				fmt.Printf("  • %s (%s/%s)\n", r.Instance.ID, r.Instance.Account, r.Instance.Region)
				if err := r.GetError(); err != nil {
					fmt.Printf("    Reason: %s\n", err)
				}
			}
		}

		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	}
}
