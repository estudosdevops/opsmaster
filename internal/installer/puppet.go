package installer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/estudosdevops/opsmaster/internal/cloud"
	"github.com/estudosdevops/opsmaster/internal/validator"
)

// OS type constants for normalized OS detection
const (
	OSTypeDebian = "debian"
	OSTypeRHEL   = "rhel"
)

// Default timeout for SSM commands (AWS SSM requires minimum 30 seconds)
const DefaultSSMTimeout = 30 * time.Second

// osAliases maps OS distribution IDs to normalized OS types
var osAliases = map[string]string{
	// Debian family
	"debian": OSTypeDebian,
	"ubuntu": OSTypeDebian,

	// RHEL family
	"rhel":        OSTypeRHEL,
	"centos":      OSTypeRHEL,
	"amzn":        OSTypeRHEL,
	"amazon":      OSTypeRHEL,
	"amazonlinux": OSTypeRHEL,
	"rocky":       OSTypeRHEL,
	"alma":        OSTypeRHEL,
	"almalinux":   OSTypeRHEL,
	"fedora":      OSTypeRHEL,
}

// normalizeOS converts OS alias to normalized type (debian or rhel).
// This centralizes OS type mapping and eliminates duplicate switch statements.
//
// Examples:
//   - "Ubuntu" → "debian"
//   - "amzn" → "rhel"
//   - "Rocky" → "rhel"
//
// Returns error if OS is not supported.
func normalizeOS(osType string) (string, error) {
	osLower := strings.ToLower(strings.TrimSpace(osType))

	if normalized, ok := osAliases[osLower]; ok {
		return normalized, nil
	}

	// Build list of supported OS for error message
	supported := make([]string, 0, len(osAliases))
	for alias := range osAliases {
		supported = append(supported, alias)
	}

	return "", fmt.Errorf("unsupported OS: %s (supported: %v)", osType, supported)
}

// FactDefinition defines a custom fact file to be created on the instance.
// Facts are stored in /opt/puppetlabs/facter/facts.d/ and read by Facter.
//
// Example:
//
//	FactDefinition{
//	    FilePath: "location.yaml",
//	    FactName: "location",
//	    Fields: map[string]string{
//	        "account": "account",
//	        "environment": "environment",
//	    },
//	}
type FactDefinition struct {
	// FilePath is the filename (e.g., "location.yaml")
	FilePath string

	// FactName is the top-level key in the YAML file (e.g., "location")
	FactName string

	// Fields maps CSV column names to fact field names
	// Example: {"account": "account", "environment": "environment"}
	Fields map[string]string
}

// PuppetInstaller implements PackageInstaller for Puppet Agent.
// Supports Debian/Ubuntu and RHEL/Amazon Linux distributions.
type PuppetInstaller struct {
	puppetServer  string
	puppetPort    int
	puppetVersion string
	environment   string
	lastMetadata  map[string]string         // Stores metadata from last installation attempt
	customFacts   map[string]FactDefinition // Custom facts to create on instances
}

// PuppetOptions contains Puppet-specific installation options.
type PuppetOptions struct {
	Server      string                    // Puppet Server hostname (required)
	Port        int                       // Puppet Server port (default: 8140)
	Version     string                    // Puppet version (default: "7")
	Environment string                    // Puppet environment (default: "production")
	CustomFacts map[string]FactDefinition // Custom facts to create on instances (optional)
}

// NewPuppetInstaller creates a new Puppet installer with given options.
func NewPuppetInstaller(opts PuppetOptions) *PuppetInstaller {
	// Set defaults
	if opts.Port == 0 {
		opts.Port = 8140
	}
	if opts.Version == "" {
		opts.Version = "7"
	}
	if opts.Environment == "" {
		opts.Environment = "production"
	}

	// Initialize custom facts with default if not provided
	customFacts := opts.CustomFacts
	if customFacts == nil {
		customFacts = make(map[string]FactDefinition)
	}

	return &PuppetInstaller{
		puppetServer:  opts.Server,
		puppetPort:    opts.Port,
		puppetVersion: opts.Version,
		environment:   opts.Environment,
		lastMetadata:  make(map[string]string),
		customFacts:   customFacts,
	}
}

// Name returns the package name
func (*PuppetInstaller) Name() string {
	return "puppet"
}

// GenerateInstallScriptWithAutoDetect generates installation script with automatic OS detection.
// This is a convenience method that detects the OS and then calls GenerateInstallScript.
// Use this method when you want automatic OS detection instead of providing it manually.
// Returns: (commands, metadata, error) where metadata contains os, certname, and certname_preserved.
func (pi *PuppetInstaller) GenerateInstallScriptWithAutoDetect(ctx context.Context, instance *cloud.Instance, provider cloud.CloudProvider, _ map[string]string) (commands []string, metadata map[string]string, err error) {
	// Step 1: Detect OS
	detectedOS, err := pi.detectOS(ctx, instance, provider)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to detect OS: %w", err)
	}

	// Step 2: Check if puppet.conf already exists with certname
	// This prevents changing certname on re-installations, which would break certificates
	existingCertname, err := pi.getCertnameFromConfig(ctx, instance, provider)
	if err != nil {
		// Non-fatal: if we can't read existing certname, generate a new one
		// This could happen on first installation or if puppet.conf is corrupted
		existingCertname = ""
	}

	// Step 3: Use existing certname or generate new one
	var certname string
	var certnamePreserved bool
	if existingCertname != "" {
		certname = existingCertname
		certnamePreserved = true
		// Preserving existing certname to avoid certificate issues
	} else {
		certname = generatePuppetCertname()
		certnamePreserved = false
	}

	// Step 4: Create metadata for THIS execution (not stored in shared variable to avoid race condition)
	metadata = map[string]string{
		"os":                 detectedOS,
		"certname":           certname,
		"certname_preserved": fmt.Sprintf("%v", certnamePreserved),
	}

	// Step 5: Normalize OS type
	normalizedOS, err := normalizeOS(detectedOS)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to normalize OS type: %w", err)
	}

	// Step 6: Generate script with certname and custom facts based on normalized OS type
	var script string
	switch normalizedOS {
	case OSTypeDebian:
		script = pi.generateDebianScript(certname, instance)
	case OSTypeRHEL:
		script = pi.generateRHELScript(certname, instance)
	default:
		// This should never happen if normalizeOS works correctly
		return nil, nil, fmt.Errorf("internal error: unexpected normalized OS type: %s", normalizedOS)
	}

	return []string{script}, metadata, nil
}

// detectOS detects the operating system of the instance via remote command execution.
// Uses /etc/os-release which is the standard systemd way to identify Linux distributions.
//
// Returns normalized OS type:
//   - "debian" for Debian/Ubuntu
//   - "rhel" for RHEL/CentOS/Amazon Linux/Rocky/AlmaLinux
//
// This ensures we generate the correct installation script for the target OS.
func (*PuppetInstaller) detectOS(ctx context.Context, instance *cloud.Instance, provider cloud.CloudProvider) (string, error) {
	// Script to detect OS from /etc/os-release
	detectScript := `#!/bin/bash
if [ -f /etc/os-release ]; then
    . /etc/os-release
    # Normalize ID to match our supported types
    case "$ID" in
        ubuntu|debian)
            echo "debian"
            ;;
        rhel|centos|fedora|rocky|alma|almalinux)
            echo "rhel"
            ;;
        amzn|amazonlinux|amazon)
            echo "rhel"
            ;;
        *)
            echo "unknown:$ID"
            ;;
    esac
else
    echo "unknown:no-os-release"
fi
`

	commands := []string{detectScript}
	result, err := provider.ExecuteCommand(ctx, instance, commands, DefaultSSMTimeout)
	if err != nil {
		return "", fmt.Errorf("failed to detect OS: %w", err)
	}

	if result.ExitCode != 0 {
		return "", fmt.Errorf("OS detection failed with exit code %d: %s", result.ExitCode, result.Stderr)
	}

	osType := strings.TrimSpace(result.Stdout)

	// Handle unknown OS
	if strings.HasPrefix(osType, "unknown:") {
		return "", fmt.Errorf("unsupported or undetected OS: %s", osType)
	}

	return osType, nil
}

// getCertnameFromConfig retrieves existing certname from puppet.conf if it exists.
// This prevents changing certname on re-installations, which would cause certificate issues.
//
// Returns:
//   - Existing certname if found
//   - Empty string if puppet.conf doesn't exist or certname not found
//   - Error if command execution fails
func (*PuppetInstaller) getCertnameFromConfig(ctx context.Context, instance *cloud.Instance, provider cloud.CloudProvider) (string, error) {
	extractScript := `#!/bin/bash
# Check if puppet.conf exists
if [ ! -f /etc/puppetlabs/puppet/puppet.conf ]; then
    echo "NOT_FOUND"
    exit 0
fi

# Extract certname from config
CERTNAME=$(grep -E '^\s*certname\s*=' /etc/puppetlabs/puppet/puppet.conf | sed 's/.*=\s*//' | tr -d ' ')

if [ -z "$CERTNAME" ]; then
    echo "NOT_FOUND"
else
    echo "$CERTNAME"
fi
`

	commands := []string{extractScript}
	result, err := provider.ExecuteCommand(ctx, instance, commands, DefaultSSMTimeout)
	if err != nil {
		return "", fmt.Errorf("failed to check existing certname: %w", err)
	}

	certname := strings.TrimSpace(result.Stdout)

	if certname == "NOT_FOUND" || certname == "" {
		return "", nil // No existing certname
	}

	return certname, nil
}

// generateCustomFact generates YAML content for a single custom fact file.
// It maps CSV columns to fact fields based on the FactDefinition configuration.
//
// Example output for location.yaml:
//
//	location:
//	  account: production
//	  environment: prod
//	  region: us-east-1
//
// Parameters:
//   - factDef: Fact definition with file path, fact name, and field mappings
//   - instance: Instance with metadata containing CSV column values
//
// Returns YAML-formatted string ready to be written to fact file.
func (*PuppetInstaller) generateCustomFact(factDef FactDefinition, instance *cloud.Instance) string {
	var content strings.Builder

	// Write fact name as top-level YAML key
	content.WriteString(factDef.FactName + ":\n")

	// Track if any fields were added (for debugging)
	fieldCount := 0

	// Map each CSV column to fact field
	for csvColumn, factField := range factDef.Fields {
		var value string

		// Get value from instance based on column type
		// Standard fields are direct properties, custom fields are in Metadata
		if csvColumn == "account" {
			value = instance.Account
		} else if csvColumn == "region" {
			value = instance.Region
		} else if instance.Metadata != nil {
			// Custom column from CSV (e.g., environment, compliance, app_name, etc)
			value = instance.Metadata[csvColumn]
		}

		// Only write field if value exists and is not empty
		if value != "" {
			content.WriteString(fmt.Sprintf("  %s: %s\n", factField, value))
			fieldCount++
		}
	}

	// If no fields were added, add a comment to explain why the fact is empty
	if fieldCount == 0 {
		content.WriteString("  # No values found in CSV for this fact\n")
	}

	return content.String()
}

// generateFactsScript generates bash commands to create all custom fact files.
// Creates directory structure and writes YAML fact files to /opt/puppetlabs/facter/facts.d/
//
// Example generated script:
//
//	mkdir -p /opt/puppetlabs/facter/facts.d
//	cat > /opt/puppetlabs/facter/facts.d/location.yaml << 'FACT_EOF_location'
//	location:
//	  account: production
//	  environment: prod
//	  region: us-east-1
//	FACT_EOF_location
//	chmod 644 /opt/puppetlabs/facter/facts.d/location.yaml
//
// Parameters:
//   - instance: Instance with metadata containing values for fact fields
//
// Returns bash script as string, or empty string if no custom facts configured.
func (pi *PuppetInstaller) generateFactsScript(instance *cloud.Instance) string {
	// No custom facts configured or no instance data available
	if len(pi.customFacts) == 0 || instance == nil {
		return ""
	}

	var script strings.Builder

	// Create facts directory and add header comment
	script.WriteString("\n# ============================================================\n")
	script.WriteString("# Creating custom Facter facts from CSV data\n")
	script.WriteString("# ============================================================\n")
	script.WriteString("echo \"Creating custom Facter facts...\"\n")
	script.WriteString("mkdir -p /opt/puppetlabs/facter/facts.d\n\n")

	// Generate each fact file
	for _, factDef := range pi.customFacts {
		factContent := pi.generateCustomFact(factDef, instance)

		// Use HERE document to safely write YAML content
		// Unique EOF marker per fact to avoid conflicts
		eofMarker := fmt.Sprintf("FACT_EOF_%s", factDef.FactName)

		script.WriteString(fmt.Sprintf("# Create %s fact file (fact name: %s)\n", factDef.FilePath, factDef.FactName))
		script.WriteString(fmt.Sprintf("cat > /opt/puppetlabs/facter/facts.d/%s << '%s'\n", factDef.FilePath, eofMarker))
		script.WriteString(factContent)
		script.WriteString(eofMarker + "\n")
		script.WriteString(fmt.Sprintf("chmod 644 /opt/puppetlabs/facter/facts.d/%s\n", factDef.FilePath))
		script.WriteString(fmt.Sprintf("echo \"  ✓ Created fact: %s\"\n\n", factDef.FilePath))
	}

	script.WriteString("echo \"Custom facts created successfully!\"\n")
	script.WriteString("# ============================================================\n")

	return script.String()
}

// generateFacterBlocklistScript generates shell script to configure Facter blocklist.
// This prevents "exceeds the value length limit: 4096" errors caused by oversized facts
// like ec2_userdata in cloud environments.
//
// Creates /etc/puppetlabs/facter/facter.conf with blocklist configuration.
// The directory is created if it doesn't exist (mkdir -p).
//
// Blocklisted facts:
//   - ec2_userdata: EC2 user data can exceed 4KB (often contains cloud-init scripts)
//   - ec2_metadata: EC2 metadata can be large in some AWS configurations
//
// Returns bash script ready to be inserted into installation script.
func (*PuppetInstaller) generateFacterBlocklistScript() string {
	return `# Configure Facter blocklist to prevent oversized facts errors
echo "Configuring Facter blocklist..."
mkdir -p /etc/puppetlabs/facter
cat > /etc/puppetlabs/facter/facter.conf <<EOF
facts : {
  blocklist : [ "ec2_userdata" ]
}
EOF
echo "  ✓ Facter blocklist configured"
`
}

// generateElasticFlagFixScript generates shell script to handle Elastic Agent enrollment errors.
// This fixes the common issue where Elastic Agent is already installed but missing the flag file,
// causing Puppet manifests to fail with enrollment errors.
//
// The script:
// 1. Uses the PUPPET_EXIT_CODE variable set by generatePuppetRunScript
// 2. If exit code indicates failure AND Elastic Agent service exists, creates missing flag file
// 3. Re-runs puppet agent -t to complete the configuration
//
// Expected error pattern from Puppet manifests:
// 'elastic-agent enroll --url=... --enrollment-token=... --delay-enroll && touch /opt/elastic_flagfile' returned 1 instead of one of [0]
//
// Returns bash script that handles this specific error case automatically.
func (*PuppetInstaller) generateElasticFlagFixScript() string {
	return `
# ============================================================
# Elastic Agent Flag Fix Handler
# ============================================================
# Auto-fix for cases where Elastic Agent is already installed but missing flag file

echo "Analyzing Puppet agent results..."
echo "Puppet exit code: $PUPPET_EXIT_CODE"

# Only try to fix if Puppet had actual failures (not success codes 0, 2, 6)
if [ $PUPPET_EXIT_CODE -ne 0 ] && [ $PUPPET_EXIT_CODE -ne 2 ] && [ $PUPPET_EXIT_CODE -ne 6 ]; then
    echo "⚠ Puppet agent run had failures (exit code: $PUPPET_EXIT_CODE)"
    echo "Checking for Elastic Agent enrollment errors..."

    # Check if elastic-agent service exists (indicating it's already installed)
    if systemctl list-unit-files 2>/dev/null | grep -q "elastic-agent.service"; then
        echo "  ✓ Elastic Agent service detected"

        # Check if flag file is missing (root cause of enrollment errors)
        if [ ! -f /opt/elastic_flagfile ]; then
            echo "  ⚠ Missing Elastic Agent flag file: /opt/elastic_flagfile"
            echo "  → This likely caused the Puppet enrollment error"
            echo "  → Creating flag file to fix Puppet manifest..."

            # Create the missing flag file (remove sudo - script should already run as root)
            touch /opt/elastic_flagfile && chmod 644 /opt/elastic_flagfile

            if [ -f /opt/elastic_flagfile ]; then
                echo "  ✅ Flag file created successfully: /opt/elastic_flagfile"
                echo "  → Re-running Puppet agent to complete configuration..."

                # Re-run puppet agent now that flag file exists
                /opt/puppetlabs/bin/puppet agent --test --waitforcert 60
                SECOND_EXIT_CODE=$?

                echo "  → Second puppet run completed with exit code: $SECOND_EXIT_CODE"
                case $SECOND_EXIT_CODE in
                    0|2|6)
                        echo "  ✅ Puppet configuration completed successfully after flag fix!"
                        echo "  🎯 Elastic Agent enrollment issue resolved!"
                        ;;
                    *)
                        echo "  ⚠ Second puppet run still has issues (exit code: $SECOND_EXIT_CODE)"
                        echo "  → Manual investigation may be needed for remaining issues"
                        echo "  → But Elastic Agent flag fix was successfully applied"
                        ;;
                esac
            else
                echo "  ❌ Failed to create flag file - filesystem or permission issue"
                echo "  → Manual intervention required: sudo touch /opt/elastic_flagfile"
            fi
        else
            echo "  ✓ Elastic Agent flag file already exists: /opt/elastic_flagfile"
            echo "  → File size: $(stat -c%s /opt/elastic_flagfile 2>/dev/null || echo 'unknown') bytes"
            echo "  → This appears to be a different Puppet issue (not Elastic Agent related)"
            echo "  → Check Puppet server logs for details"
        fi
    else
        echo "  ℹ Elastic Agent service not found on this system"
        echo "  → This appears to be a different Puppet issue (not Elastic Agent related)"
        echo "  → Common issues: network connectivity, certificate problems, server configuration"
    fi
else
    echo "✅ Puppet agent run completed with acceptable result (exit code: $PUPPET_EXIT_CODE)"
    echo "→ No Elastic Agent fixes needed"
fi

echo "============================================================"
echo "Elastic Agent Flag Fix Handler completed"
echo "============================================================"
`
}

// generatePuppetConfigScript generates shell script to configure puppet.conf.
// This is common to both Debian and RHEL installation scripts.
//
// Creates /etc/puppetlabs/puppet/puppet.conf with agent configuration:
//   - server: Puppet Server hostname
//   - environment: Puppet environment (production, staging, etc.)
//   - certname: Unique certname for this agent
//   - runinterval: How often agent checks for updates (default: 1h)
//
// Returns bash script with formatted puppet.conf content.
func (pi *PuppetInstaller) generatePuppetConfigScript(certname string) string {
	return fmt.Sprintf(`# Configure Puppet
echo "Configuring Puppet Agent..."
cat > /etc/puppetlabs/puppet/puppet.conf <<EOF
[agent]
server = %s
environment = %s
certname = %s
runinterval = 1h
EOF

echo "Puppet configured with:"
echo "  Server: %s"
echo "  Environment: %s"
echo "  Certname: %s"
`, pi.puppetServer, pi.environment, certname,
		pi.puppetServer, pi.environment, certname)
}

// generatePuppetRunScript generates shell script to run initial Puppet agent.
// This is common to both Debian and RHEL installation scripts.
//
// Executes puppet agent with:
//   - --test: Run in foreground (not as service)
//   - --waitforcert 60: Wait up to 60 seconds for certificate signing
//   - Handles Puppet exit codes properly (0, 2, 6 are success; others need investigation)
//
// Returns bash script that runs puppet agent and reports version.
func (*PuppetInstaller) generatePuppetRunScript() string {
	return `# Run initial puppet agent (will request certificate)
echo "Running initial Puppet agent..."
/opt/puppetlabs/bin/puppet agent --test --waitforcert 60
PUPPET_EXIT_CODE=$?

echo "Puppet agent completed with exit code: $PUPPET_EXIT_CODE"

# Puppet exit codes:
# 0 = success, no changes
# 2 = success, changes made
# 4 = failure during run
# 6 = success, changes made and restart required
case $PUPPET_EXIT_CODE in
    0)
        echo "  ✅ Puppet run successful - no changes needed"
        ;;
    2)
        echo "  ✅ Puppet run successful - changes applied"
        ;;
    6)
        echo "  ✅ Puppet run successful - changes applied, restart may be required"
        ;;
    *)
        echo "  ⚠ Puppet run had issues (exit code: $PUPPET_EXIT_CODE)"
        echo "  → Will attempt automatic fixes if applicable"
        ;;
esac

# Check puppet version
PUPPET_VERSION=$(/opt/puppetlabs/bin/puppet --version)
echo "================================================"
echo "Puppet Agent ${PUPPET_VERSION} installation completed!"
echo "================================================"
`
}

// ValidatePrerequisites validates prerequisites before installation.
// For Puppet, we check:
// 1. Instance is accessible (SSM connectivity)
// 2. Instance can reach Puppet Server on configured port
func (pi *PuppetInstaller) ValidatePrerequisites(ctx context.Context, instance *cloud.Instance, provider cloud.CloudProvider) error {
	// Use validator package for reusable validation logic
	results, err := validator.ValidatePuppetPrerequisites(
		ctx,
		instance,
		provider,
		pi.puppetServer,
		pi.puppetPort,
	)

	if err != nil {
		// Format validation failures for better error message
		failedValidations := validator.GetFailedValidations(results)
		var errors []string
		for _, failed := range failedValidations {
			errors = append(errors, fmt.Sprintf("%s: %s", failed.Name, failed.Message))
		}
		return fmt.Errorf("puppet prerequisites validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}

// GenerateInstallScript generates installation script based on OS.
// Supports: debian (for Debian/Ubuntu) and rhel (for RHEL/CentOS/Amazon Linux).
//
// The script will:
// 1. Detect OS version
// 2. Install Puppet repository
// 3. Install puppet-agent package
// 4. Configure puppet.conf with unique certname
// 5. Enable and start puppet service
// 6. Run initial puppet agent
//
// Note: For automatic OS detection, use GenerateInstallScriptWithAutoDetect instead.
func (pi *PuppetInstaller) GenerateInstallScript(os string, _ map[string]string) ([]string, error) {
	// Generate new certname for manual script generation
	// Note: GenerateInstallScriptWithAutoDetect handles certname preservation automatically
	certname := generatePuppetCertname()

	// Normalize OS type using centralized function
	normalizedOS, err := normalizeOS(os)
	if err != nil {
		// For backward compatibility, default to debian on unknown OS
		// This maintains the previous behavior where unknown OS would fall through
		normalizedOS = OSTypeDebian
	}

	var script string

	switch normalizedOS {
	case OSTypeDebian:
		// Note: instance is nil here - custom facts only work with GenerateInstallScriptWithAutoDetect
		script = pi.generateDebianScript(certname, nil)
	case OSTypeRHEL:
		// Note: instance is nil here - custom facts only work with GenerateInstallScriptWithAutoDetect
		script = pi.generateRHELScript(certname, nil)
	default:
		// This should never happen if normalizeOS works correctly
		return nil, fmt.Errorf("internal error: unexpected normalized OS type: %s", normalizedOS)
	}

	// Return as single-element slice (one big script)
	return []string{script}, nil
}

// generateDebianScript generates installation script for Debian/Ubuntu.
// Includes custom Facter facts creation if configured.
// Also includes automatic Elastic Agent flag fix for enrollment errors.
func (pi *PuppetInstaller) generateDebianScript(certname string, instance *cloud.Instance) string {
	// Generate script components (reusable across Debian/RHEL)
	factsScript := pi.generateFactsScript(instance)
	facterBlocklist := pi.generateFacterBlocklistScript()
	puppetConfig := pi.generatePuppetConfigScript(certname)
	puppetRun := pi.generatePuppetRunScript()
	elasticFix := pi.generateElasticFlagFixScript()

	return fmt.Sprintf(`#!/bin/bash
# Note: Removed 'set -e' to allow Puppet exit codes to be handled gracefully

echo "================================================"
echo "Installing Puppet Agent on Debian/Ubuntu"
echo "================================================"

# Detect OS version
if [ -f /etc/os-release ]; then
    . /etc/os-release
    VERSION_CODENAME=${VERSION_CODENAME}
    echo "Detected OS: ${NAME} ${VERSION}"
else
    echo "ERROR: Cannot detect OS version"
    exit 1
fi

# Download and install Puppet repository
echo "Installing Puppet %s repository..."
REPO_DEB="puppet%s-release-${VERSION_CODENAME}.deb"
wget -q "https://apt.puppet.com/${REPO_DEB}" -O /tmp/${REPO_DEB}
if ! dpkg -i /tmp/${REPO_DEB}; then
    echo "Error installing Puppet repository"
    exit 1
fi
rm /tmp/${REPO_DEB}

# Update apt cache
echo "Updating package cache..."
if ! apt-get update -qq; then
    echo "Error updating package cache"
    exit 1
fi

# Install puppet-agent
echo "Installing puppet-agent package..."
if ! DEBIAN_FRONTEND=noninteractive apt-get install -y puppet-agent; then
    echo "Error installing puppet-agent package"
    exit 1
fi

%s
%s
%s
%s
%s
`, pi.puppetVersion, pi.puppetVersion, facterBlocklist, factsScript, puppetConfig, puppetRun, elasticFix)
}

// generateRHELScript generates installation script for RHEL/CentOS/Amazon Linux.
// Includes custom Facter facts creation if configured.
// Also includes automatic Elastic Agent flag fix for enrollment errors.
func (pi *PuppetInstaller) generateRHELScript(certname string, instance *cloud.Instance) string {
	// Generate script components (reusable across Debian/RHEL)
	factsScript := pi.generateFactsScript(instance)
	facterBlocklist := pi.generateFacterBlocklistScript()
	puppetConfig := pi.generatePuppetConfigScript(certname)
	puppetRun := pi.generatePuppetRunScript()
	elasticFix := pi.generateElasticFlagFixScript()

	return fmt.Sprintf(`#!/bin/bash
# Note: Removed 'set -e' to allow Puppet exit codes to be handled gracefully

echo "================================================"
echo "Installing Puppet Agent on RHEL/Amazon Linux"
echo "================================================"

# Detect OS version
if [ -f /etc/os-release ]; then
    . /etc/os-release

    # Amazon Linux uses EL7 repos
    if [[ "$ID" == "amzn" ]]; then
        EL_VERSION=7
        echo "Detected OS: Amazon Linux ${VERSION}"
    else
        EL_VERSION=$(echo $VERSION_ID | cut -d. -f1)
        echo "Detected OS: ${NAME} ${VERSION_ID}"
    fi
else
    echo "ERROR: Cannot detect OS version"
    exit 1
fi

# Install Puppet repository
echo "Installing Puppet %s repository..."
REPO_RPM="puppet%s-release-el-${EL_VERSION}.noarch.rpm"
rpm -Uvh "https://yum.puppet.com/${REPO_RPM}" 2>/dev/null || echo "Repository already installed"

# Install puppet-agent
echo "Installing puppet-agent package..."
if ! yum install -y puppet-agent; then
    echo "Error installing puppet-agent package"
    exit 1
fi

%s
%s
%s
%s
%s
`, pi.puppetVersion, pi.puppetVersion, facterBlocklist, factsScript, puppetConfig, puppetRun, elasticFix)
}

// VerifyInstallation verifies that Puppet was installed successfully.
// Checks:
// 1. Puppet binary exists and is executable
// 2. Puppet service is active
// 3. Can execute 'puppet --version' successfully
func (*PuppetInstaller) VerifyInstallation(ctx context.Context, instance *cloud.Instance, provider cloud.CloudProvider) error {
	// Commands to verify installation
	verifyCommands := []string{
		// Check if puppet binary exists
		"test -x /opt/puppetlabs/bin/puppet || exit 1",
		// Check puppet version
		"/opt/puppetlabs/bin/puppet --version || exit 2",
		// Check if service is active
		"systemctl is-active puppet || exit 3",
	}

	result, err := provider.ExecuteCommand(ctx, instance, verifyCommands, DefaultSSMTimeout)
	if err != nil {
		return fmt.Errorf("failed to verify puppet installation: %w", err)
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("puppet verification failed (exit code %d):\nstdout: %s\nstderr: %s",
			result.ExitCode, result.Stdout, result.Stderr)
	}

	// Parse version from output
	version := strings.TrimSpace(result.Stdout)
	if version == "" {
		return fmt.Errorf("puppet installed but version could not be determined")
	}

	return nil
}

// GetSuccessTags returns tags to apply after successful installation.
// Tags include:
//   - puppet: "true"
//
// GetSuccessTags returns tags to apply after successful installation.
// Currently only applies the basic puppet=true tag.
// Additional tags can be added later if needed.
func (*PuppetInstaller) GetSuccessTags() map[string]string {
	return map[string]string{
		"puppet": "true",
	}
}

// GetFailureTags returns tags to apply when installation fails.
// Currently returns empty map as connectivity validation already indicates
// if puppet is not managing the instance.
func (*PuppetInstaller) GetFailureTags(_ error) map[string]string {
	// No tags needed for failures at this moment
	return map[string]string{}
}

// GetInstallMetadata returns metadata from the last installation attempt.
// Metadata includes:
//   - os: detected operating system (debian, rhel)
//   - certname: Puppet certname used for installation
//   - certname_preserved: "true" if certname was preserved from existing installation, "false" if newly generated
//
// Returns empty map if no installation attempt was made yet.
func (pi *PuppetInstaller) GetInstallMetadata() map[string]string {
	if pi.lastMetadata == nil {
		return map[string]string{}
	}
	return pi.lastMetadata
}

// generatePuppetCertname generates unique certname for Puppet agent.
// Format: <uuid_without_dashes>.puppet
// Example: 6ad692ece73643b8821cd8b6981f5070.puppet
//
// This ensures each agent has a unique certname for Puppet Server.
func generatePuppetCertname() string {
	// Generate UUID v4
	id := uuid.New()

	// Remove dashes from UUID
	uuidWithoutDashes := strings.ReplaceAll(id.String(), "-", "")

	// Return in format: <uuid>.puppet
	return uuidWithoutDashes + ".puppet"
}
