// cmd/nelm/rollback.go
package nelm

import (
	"time"

	"github.com/estudosdevops/opsmaster/internal/nelm"
	"github.com/spf13/cobra"
)

var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Faz rollback de uma release do nelm",
	Long: `Executa rollback de uma release para uma revisão anterior.

Exemplos:
  # Fazer rollback para a revisão anterior
  opsmaster nelm rollback -r sample-api -x kubedev

  # Fazer rollback para uma revisão específica
  opsmaster nelm rollback -r sample-api -x kubedev --revision 2

  # Com namespace customizado
  opsmaster nelm rollback -r sample-api -x kubedev -n default --revision 3

  # Com auto-approve
  opsmaster nelm rollback -r sample-api -x kubedev --auto-approve

  # Com timeout personalizado
  opsmaster nelm rollback -r sample-api -x kubedev --timeout 10m`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Usar função helper para ler flags comuns
		kubeContext, releaseName, namespace, timeoutStr, autoApprove := nelm.ReadCommonNelmFlags(cmd)
		revision, _ := cmd.Flags().GetInt("revision")

		// Validar campos obrigatórios
		if err := nelm.ValidateNelmRequiredFields(map[string]string{
			"release":     releaseName,
			"kubeContext": kubeContext,
		}); err != nil {
			return err
		}

		// Parse timeout usando função helper
		timeout := nelm.ParseTimeoutFromFlag(timeoutStr, 5*time.Minute)

		opts := nelm.RollbackOptions{
			BaseOptions: nelm.BaseOptions{
				ReleaseName: releaseName,
				Namespace:   namespace,
				KubeContext: kubeContext,
				AutoApprove: autoApprove,
				Timeout:     timeout,
			},
			Revision: revision,
		}

		// Usar função helper para logging
		nelm.LogNelmCommandStart("rollback", map[string]interface{}{
			"kubeContext": kubeContext,
			"release":     releaseName,
			"namespace":   namespace,
			"revision":    revision,
			"autoApprove": autoApprove,
			"timeout":     timeout,
		})

		if err := nelm.ExecuteSmartRollback(opts); err != nil {
			nelm.LogNelmCommandError("rollback", err)
			return err
		}
		nelm.LogNelmCommandSuccess("rollback")
		return nil
	},
}

func init() {
	// Flags obrigatórias
	rollbackCmd.Flags().StringP("release", "r", "", "Nome da release para rollback (obrigatório)")
	rollbackCmd.Flags().StringP("kube-context", "x", "", "Contexto do kubeconfig a ser usado (obrigatório)")

	// Flags opcionais
	rollbackCmd.Flags().Int("revision", 0, "Número da revisão para rollback (0 = revisão anterior)")
	rollbackCmd.Flags().Bool("auto-approve", false, "Pular confirmação interativa")
	rollbackCmd.Flags().String("timeout", "5m", "Timeout para a operação (ex: 10s, 1m, 1h)")

	rollbackCmd.MarkFlagRequired("release")
	rollbackCmd.MarkFlagRequired("kube-context")
}
