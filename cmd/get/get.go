// cmd/get/get.go
package get

import (
	"github.com/spf13/cobra"
)

// GetCmd é o comando pai "get". É exportado para que o pacote raiz (cmd)
// possa encontrá-lo e adicioná-lo.
var GetCmd = &cobra.Command{
	Use:   "get",
	Short: "Busca e exibe diferentes tipos de recursos",
	Long:  `O comando 'get' é um agrupador para subcomandos que buscam e exibem informações de vários recursos, como IP público, registros DNS, etc.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// A função init() adiciona os comandos filhos a este grupo.
func init() {
	GetCmd.AddCommand(ipCmd)
	GetCmd.AddCommand(dnsCmd)
}
