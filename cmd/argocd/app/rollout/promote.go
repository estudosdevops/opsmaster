// cmd/argocd/app/rollout/promote.go
package rollout

import (
	"time"

	"github.com/estudosdevops/opsmaster/internal/argocd"
	"github.com/spf13/cobra"
)

var (
	promoteWait        bool
	promoteWaitTimeout time.Duration
)

var promoteCmd = &cobra.Command{
	Use:   "promote <nome-da-aplicacao>",
	Short: "Promove o rollout de uma aplicação para a próxima etapa",
	Long:  `Envia um comando para o Argo Rollouts para avançar o deploy de uma aplicação para a próxima etapa definida na sua estratégia.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return handleRolloutAction(cmd, args, argocd.PromoteApplicationRollout, "promote", promoteWait, promoteWaitTimeout)
	},
}

func init() {
	promoteCmd.Flags().BoolVar(&promoteWait, "wait", false, "Aguarda a aplicação ficar saudável e sincronizada após a promoção")
	promoteCmd.Flags().DurationVar(&promoteWaitTimeout, "wait-timeout", 5*time.Minute, "Tempo máximo de espera pela aplicação")
}
