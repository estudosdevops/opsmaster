// cmd/argocd/app/sync.go
package app

import (
	"context"
	"time"

	"github.com/estudosdevops/opsmaster/internal/argocd"
	"github.com/estudosdevops/opsmaster/internal/logger"
	"github.com/spf13/cobra"
)

var forceSync bool

// appSyncCmd representa o comando "opsmaster argocd app sync".
var appSyncCmd = &cobra.Command{
	Use:   "sync <nome-da-aplicacao>",
	Short: "Força a sincronização de uma aplicação no Argo CD",
	Long:  `Inicia uma sincronização imediata para uma aplicação, fazendo com que ela corresponda ao estado definido no repositório Git.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		log := logger.Get()
		appName := args[0]
		serverAddr, _ := cmd.Flags().GetString("server")
		authToken, _ := cmd.Flags().GetString("token")
		insecure, _ := cmd.Flags().GetBool("insecure")

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		logMessage := "Iniciando sincronização da aplicação..."
		if forceSync {
			logMessage = "Forçando a sincronização da aplicação..."
		}
		log.Info(logMessage, "aplicação", appName, "force", forceSync)

		// Chama a nossa nova lógica para sincronizar a aplicação, passando o valor da flag 'force'.
		if err := argocd.SyncApplication(ctx, serverAddr, authToken, insecure, appName, forceSync); err != nil {
			log.Error("Falha ao sincronizar a aplicação", "erro", err)
			return err
		}

		log.Info("Pedido de sincronização enviado com sucesso!", "aplicação", appName)
		log.Info("Use 'opsmaster argocd app wait' para aguardar a conclusão da sincronização.")
		return nil
	},
}

func init() {
	appSyncCmd.Flags().BoolVar(&forceSync, "force", false, "Força a sincronização, substituindo recursos e apagando os que não existem mais no Git (prune)")
}
