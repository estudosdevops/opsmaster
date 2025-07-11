// cmd/argocd/app/rollout/helper.go
package rollout

import (
	"context"
	"time"

	"github.com/estudosdevops/opsmaster/internal/argocd"
	"github.com/estudosdevops/opsmaster/internal/logger"
	"github.com/spf13/cobra"
)

type rolloutActionFunc func(ctx context.Context, serverAddr, authToken string, insecure bool, appName string) error

func handleRolloutAction(cmd *cobra.Command, args []string, actionFunc rolloutActionFunc, actionName string, wait bool, waitTimeout time.Duration) error {
	log := logger.Get()
	appName := args[0]
	serverAddr, _ := cmd.Flags().GetString("server")
	authToken, _ := cmd.Flags().GetString("token")
	insecure, _ := cmd.Flags().GetBool("insecure")

	actionCtx, actionCancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer actionCancel()

	log.Info("Enviando comando de '"+actionName+"' para o rollout...", "aplicação", appName)
	if err := actionFunc(actionCtx, serverAddr, authToken, insecure, appName); err != nil {
		log.Error("Falha ao executar a ação de rollout", "ação", actionName, "erro", err)
		return err
	}
	log.Info("Comando '"+actionName+"' enviado com sucesso!", "aplicação", appName)

	if wait {
		waitCtx, waitCancel := context.WithTimeout(context.Background(), waitTimeout)
		defer waitCancel()
		waitInterval := 15 * time.Second
		if _, err := argocd.WaitForAppStatus(waitCtx, serverAddr, authToken, insecure, appName, waitInterval); err != nil {
			log.Error("A aplicação não atingiu o estado desejado após a ação", "ação", actionName, "erro", err)
			return err
		}
	}
	return nil
}
