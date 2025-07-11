package rollout

import (
	"time"

	"github.com/estudosdevops/opsmaster/internal/argocd"
	"github.com/spf13/cobra"
)

var (
	abortWait        bool
	abortWaitTimeout time.Duration
)

var abortCmd = &cobra.Command{
	Use:   "abort <nome-da-aplicacao>",
	Short: "Aborta um rollout em andamento",
	Long:  `Envia um comando para o Argo Rollouts para cancelar um deploy em andamento e reverter para a versão estável anterior.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return handleRolloutAction(
			cmd,
			args,
			argocd.AbortApplicationRollout,
			"abort",
			abortWait,
			abortWaitTimeout,
		)
	},
}

func init() {
	abortCmd.Flags().BoolVar(&abortWait, "wait", false, "Aguarda a aplicação ficar saudável e sincronizada após o abort")
	abortCmd.Flags().DurationVar(&abortWaitTimeout, "wait-timeout", 5*time.Minute, "Tempo máximo de espera pela aplicação")
}
