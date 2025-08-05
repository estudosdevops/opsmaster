// cmd/nelm/nelm.go
package nelm

import (
	"github.com/spf13/cobra"
)

// NelmCmd representa o comando pai "nelm".
var NelmCmd = &cobra.Command{
	Use:   "nelm",
	Short: "Executa comandos 'nelm' de forma inteligente",
	Long:  `Um conjunto de comandos para orquestrar a ferramenta 'nelm', simplificando a gestão de releases em ambientes de CI/CD.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	NelmCmd.AddCommand(installCmd)
	NelmCmd.AddCommand(uninstallCmd)
	NelmCmd.AddCommand(statusCmd)
	NelmCmd.AddCommand(rollbackCmd)
}
