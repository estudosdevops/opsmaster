// cmd/nelm/status.go
package nelm

import (
	"time"

	"github.com/estudosdevops/opsmaster/internal/nelm"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Verifica o status de releases do nelm",
	Long: `Verifica o status de releases instaladas no nelm.

Exemplos:
  # Verificar status de uma release específica
  opsmaster nelm status -r sample-api -x kubedev

  # Com namespace customizado
  opsmaster nelm status -r sample-api -x kubedev -n default

  # Com timeout personalizado
  opsmaster nelm status -r sample-api -x kubedev --timeout 10m`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Usar função helper para ler flags comuns
		kubeContext, releaseName, namespace, timeoutStr, _ := nelm.ReadCommonNelmFlags(cmd)

		// Validar campos obrigatórios
		if err := nelm.ValidateNelmRequiredFields(map[string]string{
			"release":     releaseName,
			"kubeContext": kubeContext,
		}); err != nil {
			return err
		}

		// Parse timeout usando função helper
		timeout := nelm.ParseTimeoutFromFlag(timeoutStr, 5*time.Minute)

		opts := nelm.StatusOptions{
			BaseOptions: nelm.BaseOptions{
				ReleaseName: releaseName,
				Namespace:   namespace,
				KubeContext: kubeContext,
				Timeout:     timeout,
			},
		}

		// Usar função helper para logging
		nelm.LogNelmCommandStart("verificação de status", map[string]interface{}{
			"kubeContext": kubeContext,
			"release":     releaseName,
			"namespace":   namespace,
			"timeout":     timeout,
		})

		if err := nelm.ExecuteSmartStatus(opts); err != nil {
			nelm.LogNelmCommandError("verificação de status", err)
			return err
		}
		nelm.LogNelmCommandSuccess("verificação de status")
		return nil
	},
}

func init() {
	// Flags para o comando status
	statusCmd.Flags().StringP("release", "r", "", "Nome da release específica a ser verificada (obrigatório)")
	statusCmd.Flags().StringP("kube-context", "x", "", "Contexto do kubeconfig a ser usado (obrigatório)")
	statusCmd.Flags().StringP("namespace", "n", "", "Namespace onde buscar releases (opcional)")
	statusCmd.Flags().String("timeout", "5m", "Timeout para a operação (ex: 10s, 1m, 1h)")

	statusCmd.MarkFlagRequired("release")
	statusCmd.MarkFlagRequired("kube-context")
}
