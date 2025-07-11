// cmd/argocd/app/rollout/rollout.go
package rollout

import (
	"github.com/spf13/cobra"
)

// RolloutCmd é o comando "rollout" dentro do grupo "app". É exportado.
var RolloutCmd = &cobra.Command{
	Use:   "rollout",
	Short: "Gerencia rollouts de uma aplicação",
	Long:  `Um conjunto de subcomandos para gerenciar o ciclo de vida de um Argo Rollout associado a uma aplicação.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	// Adiciona todos os comandos de ação de rollout a este grupo.
	RolloutCmd.AddCommand(promoteCmd)
	RolloutCmd.AddCommand(abortCmd)
	RolloutCmd.AddCommand(retryCmd)
}
