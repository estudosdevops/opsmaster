package install

import (
	"github.com/spf13/cobra"
)

// InstallCmd represents the install command
// This is the root command for all installation operations
// Usage: opsmaster install <package> [flags]
var InstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Instala pacotes em instâncias na nuvem",
	Long: `Instala pacotes (Puppet, Docker, etc) em múltiplas instâncias na nuvem em paralelo.

Suporta múltiplos provedores de nuvem (AWS, Azure, GCP) e pacotes.
Utiliza execução remota (SSM para AWS) para instalar e configurar pacotes.

Exemplos:
  # Instalar Puppet em instâncias a partir de arquivo CSV
  opsmaster install puppet --instances-file instances.csv --puppet-server puppet.example.com

  # Instalar com concorrência customizada
  opsmaster install puppet --instances-file instances.csv --puppet-server puppet.example.com --max-concurrency 20

  # Modo dry run (simular sem executar)
  opsmaster install puppet --instances-file instances.csv --puppet-server puppet.example.com --dry-run`,

	// No Run function - this is just a parent command
	// Actual work is done by subcommands (puppet, docker, etc)
}

func init() {
	// Add subcommands here
	// InstallCmd.AddCommand(puppetCmd)
	// InstallCmd.AddCommand(dockerCmd)  // Future
}
