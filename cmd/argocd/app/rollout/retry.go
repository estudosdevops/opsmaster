package rollout

import (
	"time"

	"github.com/estudosdevops/opsmaster/internal/argocd"
	"github.com/spf13/cobra"
)

var (
	retryWait        bool
	retryWaitTimeout time.Duration
)

var retryCmd = &cobra.Command{
	Use:   "retry <nome-da-aplicacao>",
	Short: "Tenta novamente uma etapa de um rollout que falhou",
	Long:  `Envia um comando para o Argo Rollouts para tentar executar novamente a última etapa de um rollout que resultou em falha.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return handleRolloutAction(
			cmd,
			args,
			argocd.RetryApplicationRollout,
			"retry",
			retryWait,
			retryWaitTimeout,
		)
	},
}

func init() {
	retryCmd.Flags().BoolVar(&retryWait, "wait", false, "Aguarda a aplicação ficar saudável e sincronizada após a nova tentativa")
	retryCmd.Flags().DurationVar(&retryWaitTimeout, "wait-timeout", 5*time.Minute, "Tempo máximo de espera pela aplicação")
}
