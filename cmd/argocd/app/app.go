// cmd/argocd/app/app.go
package app

import (
	"github.com/spf13/cobra"
)

// AppCmd é o comando "app" dentro do grupo "argocd".
var AppCmd = &cobra.Command{
	Use:   "app",
	Short: "Gerencia aplicações do Argo CD",
	Long:  `Um conjunto de subcomandos para criar, listar e aguardar o status de aplicações no Argo CD.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	// Adiciona os comandos de ação ('create', 'list', 'wait') a este grupo.
	AppCmd.AddCommand(appCreateCmd)
	AppCmd.AddCommand(appListCmd)
	AppCmd.AddCommand(appWaitCmd)
}
