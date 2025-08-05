// cmd/nelm/uninstall.go
package nelm

import (
	"time"

	"github.com/estudosdevops/opsmaster/internal/nelm"
	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Desinstala uma release do nelm",
	Long: `Executa um fluxo completo para remover uma release do nelm.

Exemplos:
  # Desinstalar uma release específica
  opsmaster nelm uninstall -r sample-api -x kubedev

  # Com namespace customizado
  opsmaster nelm uninstall -r sample-api -x kubedev -n default

  # Com auto-approve
  opsmaster nelm uninstall -r sample-api -x kubedev --auto-approve

  # Com timeout personalizado
  opsmaster nelm uninstall -r sample-api -x kubedev --timeout 10m`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Usar função helper para ler flags comuns
		kubeContext, releaseName, namespace, timeoutStr, autoApprove := nelm.ReadCommonNelmFlags(cmd)

		// Validar campos obrigatórios
		if err := nelm.ValidateNelmRequiredFields(map[string]string{
			"release":     releaseName,
			"kubeContext": kubeContext,
		}); err != nil {
			return err
		}

		// Parse timeout usando função helper
		timeout := nelm.ParseTimeoutFromFlag(timeoutStr, 5*time.Minute)

		opts := nelm.UninstallOptions{
			BaseOptions: nelm.BaseOptions{
				ReleaseName: releaseName,
				Namespace:   namespace,
				KubeContext: kubeContext,
				AutoApprove: autoApprove,
				Timeout:     timeout,
			},
		}

		// Usar função helper para logging
		nelm.LogNelmCommandStart("desinstalação", map[string]interface{}{
			"kubeContext": kubeContext,
			"release":     releaseName,
			"namespace":   namespace,
			"autoApprove": autoApprove,
			"timeout":     timeout,
		})

		if err := nelm.ExecuteSmartUninstall(opts); err != nil {
			nelm.LogNelmCommandError("desinstalação", err)
			return err
		}
		nelm.LogNelmCommandSuccess("desinstalação")
		return nil
	},
}

func init() {
	// A flag 'release' agora é opcional.
	uninstallCmd.Flags().StringP("release", "r", "", "Nome da release específica a ser desinstalada (opcional)")
	// A flag 'namespace' é opcional - se não fornecida, usa o nome da release como namespace.
	uninstallCmd.Flags().StringP("namespace", "n", "", "Namespace onde a release está instalada (opcional - usa o nome da release se não fornecido)")

	uninstallCmd.Flags().StringP("kube-context", "x", "", "Contexto do kubeconfig a ser usado (obrigatório)")
	uninstallCmd.Flags().Bool("auto-approve", false, "Pula a confirmação interativa")
	uninstallCmd.Flags().String("timeout", "5m", "Timeout para a operação (ex: 10s, 1m, 1h)")

	uninstallCmd.MarkFlagRequired("kube-context")
}
