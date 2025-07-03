// cmd/scan/scan.go
package scan

import (
	"github.com/spf13/cobra"
)

// ScanCmd é o comando pai "scan". É exportado para que o pacote raiz (cmd)
// possa encontrá-lo e adicioná-lo.
var ScanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Executa vários tipos de escaneamento em alvos de rede",
	Long:  `O comando 'scan' é um agrupador para subcomandos que realizam verificações ativas em alvos de rede, como escaneamento de portas e monitoramento de URLs.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// A função init() agora só precisa de adicionar os seus próprios filhos.
func init() {
	ScanCmd.AddCommand(portsCmd)
	ScanCmd.AddCommand(monitorCmd)
}
