// cmd/argocd/app/wait.go
package app

import (
	"context"
	"time"

	"github.com/estudosdevops/opsmaster/internal/argocd"
	"github.com/estudosdevops/opsmaster/internal/logger"
	"github.com/spf13/cobra"
)

var (
	waitTimeout  time.Duration
	waitInterval time.Duration
)

var appWaitCmd = &cobra.Command{
	Use:   "wait <nome-da-aplicacao>",
	Short: "Aguarda uma aplicação ficar saudável e sincronizada",
	Long:  `Monitora continuamente uma aplicação no Argo CD e encerra com sucesso quando o status de saúde for 'Healthy' e o status de sincronização for 'Synced'.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		log := logger.Get()
		appName := args[0]
		serverAddr, _ := cmd.Flags().GetString("server")
		authToken, _ := cmd.Flags().GetString("token")
		insecure, _ := cmd.Flags().GetBool("insecure")

		ctx, cancel := context.WithTimeout(context.Background(), waitTimeout)
		defer cancel()

		err := argocd.WaitForAppStatus(ctx, serverAddr, authToken, insecure, appName, waitInterval)
		if err != nil {
			log.Error("A aplicação não atingiu o estado desejado a tempo", "erro", err)
			return err
		}
		log.Info("Aplicação está saudável e sincronizada!", "aplicação", appName)
		return nil
	},
}

func init() {
	appWaitCmd.Flags().DurationVarP(&waitTimeout, "timeout", "t", 5*time.Minute, "Tempo máximo de espera pela aplicação")
	appWaitCmd.Flags().DurationVarP(&waitInterval, "interval", "i", 15*time.Second, "Intervalo entre as verificações de status")
}
