package install

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/estudosdevops/opsmaster/internal/cloud"
	"github.com/estudosdevops/opsmaster/internal/cloud/provider"
	"github.com/estudosdevops/opsmaster/internal/csv"
	"github.com/estudosdevops/opsmaster/internal/executor"
	"github.com/estudosdevops/opsmaster/internal/installer"
	"github.com/estudosdevops/opsmaster/internal/logger"
	"github.com/estudosdevops/opsmaster/internal/presenter"
	"github.com/estudosdevops/opsmaster/internal/retry"
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

	// Retry configuration flags
	maxRetries  int           // Maximum retry attempts for all operations
	retryDelay  time.Duration // Base delay between retries
	retryJitter bool          // Add random jitter to retry delays
	ssmRetries  int           // Maximum retry attempts for SSM operations (0 = use maxRetries)
	ec2Retries  int           // Maximum retry attempts for EC2 operations (0 = use maxRetries)
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
  Formato básico (obrigatório):
    instance_id,account,region,environment
    i-0123456789abcdef0,111111111111,us-east-1,production
    i-fedcba9876543210,111111111111,us-west-2,staging

  Formato com AWS Profile (para SSO):
    instance_id,account,region,environment,aws_profile
    i-0123456789abcdef0,111111111111,us-east-1,production,aws-staging-applications
    i-fedcba9876543210,111111111111,us-west-2,staging,aws-staging-applications

O arquivo CSV deve ter cabeçalhos: instance_id, account, region
Colunas opcionais:
  - cloud (padrão aws)
  - environment
  - aws_profile (para autenticação SSO)
  - quaisquer colunas extras são armazenadas como metadados

Autenticação AWS:
  O OpsMaster suporta três métodos de autenticação (em ordem de prioridade):
  1. Flag --aws-profile (maior prioridade)
  2. Coluna aws_profile no CSV
  3. Account ID como profile (compatibilidade com versões anteriores)

  Para usar SSO, configure profiles em ~/.aws/config e use:
    - Flag: --aws-profile nome-do-profile
    - CSV: coluna aws_profile com nome do profile por instância

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

  # Com SSO usando flag
  opsmaster install puppet \
    --instances-file instances.csv \
    --puppet-server puppet.example.com \
    --aws-profile aws-staging-applications

  # Com SSO usando CSV (csv deve ter coluna aws_profile)
  opsmaster install puppet \
    --instances-file instances-with-profiles.csv \
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

	// Retry configuration flags
	puppetCmd.Flags().IntVar(&maxRetries, "max-retries", 3, "Maximum retry attempts for operations")
	puppetCmd.Flags().DurationVar(&retryDelay, "retry-delay", 2*time.Second, "Base delay between retries")
	puppetCmd.Flags().BoolVar(&retryJitter, "retry-jitter", true, "Add random jitter to retry delays")
	puppetCmd.Flags().IntVar(&ssmRetries, "ssm-retries", 0, "Max retries for SSM operations (0 = use --max-retries)")
	puppetCmd.Flags().IntVar(&ec2Retries, "ec2-retries", 0, "Max retries for EC2 operations (0 = use --max-retries)")
}

// createPuppetRetryPolicies creates retry policies based on command line flags.
// This function implements the override hierarchy: specific flags > general flags > defaults.
//
// For Puppet operations, we optimize policies for different operation types:
// - SSM operations: Conservative (Puppet installations can be slow)
// - EC2 operations: Aggressive (EC2 APIs are fast and reliable)
//
// Returns:
//   - retry.RetryConfig: SSM policy for command execution and validation
//   - retry.RetryConfig: EC2 policy for tagging and metadata operations
func createPuppetRetryPolicies() (retry.RetryConfig, retry.RetryConfig) {
	// Validate retry configuration
	if maxRetries < 1 || maxRetries > 20 {
		// Log warning but don't fail - use default
		logger.Get().Warn("Invalid --max-retries value, using default",
			"provided", maxRetries,
			"default", 3,
			"valid_range", "1-20")
		maxRetries = 3
	}

	if retryDelay < 0 || retryDelay > 60*time.Second {
		// Log warning but don't fail - use default
		logger.Get().Warn("Invalid --retry-delay value, using default",
			"provided", retryDelay,
			"default", "2s",
			"valid_range", "0s-60s")
		retryDelay = 2 * time.Second
	}

	// SSM Policy (command execution, validation)
	// Priority: --ssm-retries > --max-retries > default
	ssmMaxAttempts := maxRetries
	if ssmRetries > 0 {
		if ssmRetries > 20 {
			logger.Get().Warn("Invalid --ssm-retries value, using --max-retries",
				"provided", ssmRetries,
				"fallback", maxRetries)
		} else {
			ssmMaxAttempts = ssmRetries
		}
	}

	ssmPolicy := retry.RetryConfig{
		MaxAttempts: ssmMaxAttempts,
		BaseDelay:   retryDelay,
		MaxDelay:    retryDelay * 30, // Puppet can take time, allow longer delays
		Jitter:      retryJitter,
	}

	// EC2 Policy (tagging, metadata)
	// Priority: --ec2-retries > --max-retries > default
	ec2MaxAttempts := maxRetries
	if ec2Retries > 0 {
		if ec2Retries > 20 {
			logger.Get().Warn("Invalid --ec2-retries value, using --max-retries",
				"provided", ec2Retries,
				"fallback", maxRetries)
		} else {
			ec2MaxAttempts = ec2Retries
		}
	}

	ec2Policy := retry.RetryConfig{
		MaxAttempts: ec2MaxAttempts,
		BaseDelay:   retryDelay / 2, // EC2 APIs are faster, use shorter delays
		MaxDelay:    retryDelay * 5, // Keep max delay reasonable for EC2
		Jitter:      retryJitter,
	}

	// Log the final retry configuration for observability
	logger.Get().Info("Retry configuration created",
		"ssm_max_attempts", ssmPolicy.MaxAttempts,
		"ssm_base_delay", ssmPolicy.BaseDelay,
		"ssm_max_delay", ssmPolicy.MaxDelay,
		"ec2_max_attempts", ec2Policy.MaxAttempts,
		"ec2_base_delay", ec2Policy.BaseDelay,
		"ec2_max_delay", ec2Policy.MaxDelay,
		"jitter", retryJitter,
	)

	return ssmPolicy, ec2Policy
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

	// Determine AWS profile to use (from flag or CSV)
	effectiveAWSProfile, err := determineAWSProfile(log, instances, awsProfile)
	if err != nil {
		return fatalError(log, "Failed to determine AWS profile", err)
	}

	// Create provider using factory
	var providerOptions []provider.Option
	if effectiveAWSProfile != "" {
		providerOptions = append(providerOptions, provider.WithProfile(effectiveAWSProfile))
		log.Info("   Using AWS profile", "profile", effectiveAWSProfile)
	}

	// Add custom retry policies if any retry flags were used
	if cmd.Flags().Changed("max-retries") || cmd.Flags().Changed("retry-delay") ||
		cmd.Flags().Changed("retry-jitter") || cmd.Flags().Changed("ssm-retries") ||
		cmd.Flags().Changed("ec2-retries") {

		// Create custom retry policies based on flags
		ssmPolicy, ec2Policy := createPuppetRetryPolicies()
		providerOptions = append(providerOptions, provider.WithRetryPolicies(ssmPolicy, ec2Policy))

		log.Info("   Using custom retry policies",
			"ssm_max_attempts", ssmPolicy.MaxAttempts,
			"ec2_max_attempts", ec2Policy.MaxAttempts,
		)
	} else {
		log.Info("   Using default retry policies")
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

// prepareResultRows converts AggregatedResult to table rows for presenter.PrintTable.
// Returns header ([]string) and rows ([][]string) with formatted data.
//
// Columns:
//   - Instance ID: Instance identifier
//   - Account: AWS account number
//   - Region: AWS region
//   - Status: ✅ Success, ❌ Failed, ⏭️ Skipped
//   - Certname: Puppet certname (with (*) if preserved)
//   - OS: Operating system (debian/rhel)
//   - Duration: Installation time in human-readable format
//   - Error: Error message for failed installations (empty for success)
func prepareResultRows(result *executor.AggregatedResult) (header []string, rows [][]string) {
	// Define headers (UPPERCASE for professional look)
	header = []string{
		"INSTANCE ID",
		"ACCOUNT",
		"REGION",
		"STATUS",
		"CERTNAME",
		"OS",
		"DURATION",
		"ERROR",
	}

	// Convert results to rows
	rows = [][]string{}
	for _, r := range result.Results {
		row := []string{
			r.Instance.ID,
			r.Instance.Account,
			r.Instance.Region,
			getStatusEmoji(r.Status),
			getCertnameDisplay(r.Metadata),
			r.Metadata["os"],
			formatDuration(r.Duration),
			formatError(r),
		}
		rows = append(rows, row)
	}

	return header, rows
}

// getStatusEmoji returns emoji representation of execution status.
func getStatusEmoji(status executor.ExecutionStatus) string {
	switch status {
	case executor.StatusSuccess:
		return "✅"
	case executor.StatusFailed:
		return "❌"
	case executor.StatusSkipped:
		return "⏭️"
	default:
		return "❓"
	}
}

// getCertnameDisplay formats certname with (*) marker if preserved.
func getCertnameDisplay(metadata map[string]string) string {
	certname := metadata["certname"]
	if certname == "" {
		return "-"
	}

	// Add marker if certname was preserved
	if metadata["certname_preserved"] == "true" {
		return certname + " (*)"
	}

	return certname
}

// formatDuration formats duration in human-readable format (45s, 1m30s, etc.)
func formatDuration(d time.Duration) string {
	if d == 0 {
		return "-"
	}

	seconds := int(d.Seconds())
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}

	minutes := seconds / 60
	seconds = seconds % 60
	if seconds == 0 {
		return fmt.Sprintf("%dm", minutes)
	}

	return fmt.Sprintf("%dm%ds", minutes, seconds)
}

// formatError formats error message for table display.
// Returns empty string for successful installations, first line of error for failed ones.
// Multi-line errors are truncated to first line for table compactness.
func formatError(r *executor.ExecutionResult) string {
	// Success/Skipped = no error message
	if r.Status != executor.StatusFailed {
		return ""
	}

	// Get error
	err := r.GetError()
	if err == nil {
		return "Unknown error"
	}

	// Extract first line only (for table readability)
	errMsg := err.Error()
	lines := strings.Split(errMsg, "\n")
	if len(lines) > 0 && strings.TrimSpace(lines[0]) != "" {
		return strings.TrimSpace(lines[0])
	}

	return errMsg
}

// printResults prints detailed results to console
func printResults(result *executor.AggregatedResult) {
	if len(result.Results) == 0 {
		return
	}

	// Print header
	fmt.Println("\n# DETAILED RESULTS:")

	// Prepare data for table
	header, rows := prepareResultRows(result)

	// Render table with borders using presenter
	presenter.PrintTable(header, rows)

	// Print footnotes
	if hasCertnamePreserved(result) {
		fmt.Println("\n(*) Certname preserved from previous installation")
	}

	// Print summary
	printSummary(result)
}

// hasCertnamePreserved checks if any instance had certname preserved.
func hasCertnamePreserved(result *executor.AggregatedResult) bool {
	for _, r := range result.Results {
		if r.Metadata["certname_preserved"] == "true" {
			return true
		}
	}
	return false
}

// printSummary prints execution summary with counts and total duration.
func printSummary(result *executor.AggregatedResult) {
	successCount := 0
	failedCount := 0
	skippedCount := 0

	for _, r := range result.Results {
		switch r.Status {
		case executor.StatusSuccess:
			successCount++
		case executor.StatusFailed:
			failedCount++
		case executor.StatusSkipped:
			skippedCount++
		}
	}

	fmt.Printf("\n📊 Summary: %d successful, %d failed, %d skipped\n",
		successCount, failedCount, skippedCount)
}

// determineAWSProfile determines which AWS profile to use for authentication.
// Priority order:
//  1. Flag --aws-profile (highest priority)
//  2. aws_profile column from CSV (per-instance)
//  3. Account ID as profile (backward compatibility fallback)
//
// Returns error if instances have conflicting profiles in CSV.
func determineAWSProfile(log *slog.Logger, instances []*cloud.Instance, flagProfile string) (string, error) {
	// If flag is provided, use it (overrides CSV)
	if flagProfile != "" {
		log.Info("   Using AWS profile from --aws-profile flag", "profile", flagProfile)
		return flagProfile, nil
	}

	// Check if CSV has aws_profile column
	var csvProfiles []string
	var instancesWithProfile []*cloud.Instance

	for _, instance := range instances {
		if profile := instance.Metadata["aws_profile"]; profile != "" {
			csvProfiles = append(csvProfiles, profile)
			instancesWithProfile = append(instancesWithProfile, instance)
		}
	}

	// If no aws_profile in CSV, fallback to account ID (backward compatibility)
	if len(csvProfiles) == 0 {
		log.Info("   No aws_profile in CSV, using account ID as profile (backward compatibility)")
		if len(instances) > 0 {
			return instances[0].Account, nil // All instances should have same account from DetectCloudFromInstances
		}
		return "", nil
	}

	// Validate all instances have the same profile
	firstProfile := csvProfiles[0]
	for i, profile := range csvProfiles {
		if profile != firstProfile {
			return "", fmt.Errorf("instances have conflicting AWS profiles: instance %s uses '%s' but instance %s uses '%s'. All instances must use the same AWS profile",
				instancesWithProfile[0].ID, firstProfile,
				instancesWithProfile[i].ID, profile)
		}
	}

	// Check if some instances have profile and others don't
	if len(instancesWithProfile) != len(instances) {
		return "", fmt.Errorf("inconsistent aws_profile usage: %d instances have aws_profile but %d don't. Either all instances must have aws_profile column or none",
			len(instancesWithProfile), len(instances)-len(instancesWithProfile))
	}

	log.Info("   Using AWS profile from CSV", "profile", firstProfile, "instances", len(instances))
	return firstProfile, nil
}
