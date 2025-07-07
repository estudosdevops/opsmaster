// cmd/argocd/app/delete.go
package app

import (
	"context"
	"time"

	"github.com/estudosdevops/opsmaster/internal/argocd"
	"github.com/estudosdevops/opsmaster/internal/logger"
	"github.com/spf13/cobra"
)

// appDeleteCmd representa o comando "opsmaster argocd app delete".
var appDeleteCmd = &cobra.Command{
	Use:   "delete <nome-da-aplicacao>",
	Short: "Apaga uma aplicação do Argo CD",
	Long:  `Remove uma aplicação do Argo CD. Por padrão, esta operação não apaga os recursos no cluster Kubernetes.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		log := logger.Get()
		appName := args[0]
		serverAddr, _ := cmd.Flags().GetString("server")
		authToken, _ := cmd.Flags().GetString("token")
		insecure, _ := cmd.Flags().GetBool("insecure")

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		log.Info("A apagar a aplicação do Argo CD...", "aplicação", appName)

		// Chama a nossa nova lógica para apagar a aplicação.
		if err := argocd.DeleteApplication(ctx, serverAddr, authToken, insecure, appName); err != nil {
			log.Error("Falha ao apagar a aplicação", "erro", err)
			return err
		}

		log.Info("Aplicação apagada com sucesso!", "aplicação", appName)
		return nil
	},
}
