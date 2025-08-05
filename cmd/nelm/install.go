// cmd/nelm/install.go
package nelm

import (
	"time"

	"github.com/estudosdevops/opsmaster/internal/nelm"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Instala releases do nelm",
	Long: `Executa um fluxo completo (chart lint, plan install, install) para instalar releases do nelm.

Exemplos:
  # Instalar todas as releases detectadas
  opsmaster nelm install --env stg -x kubedev

  # Instalar uma release específica
  opsmaster nelm install -r sample-api --env stg -x kubedev

  # Com namespace customizado
  opsmaster nelm install -r sample-api --env stg -x kubedev -n default

  # Com auto-approve
  opsmaster nelm install -r sample-api --env stg -x kubedev --auto-approve

  # Com timeout e concorrência personalizados
  opsmaster nelm install -r sample-api --env stg -x kubedev --timeout 10m --max-concurrency 5`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Usar função helper para ler flags comuns
		kubeContext, releaseName, namespace, timeoutStr, autoApprove := nelm.ReadCommonNelmFlags(cmd)
		env, _ := cmd.Flags().GetString("env")

		// Validar campos obrigatórios
		if err := nelm.ValidateNelmRequiredFields(map[string]string{
			"env":         env,
			"kubeContext": kubeContext,
		}); err != nil {
			return err
		}

		// Parse timeout usando função helper
		timeout := nelm.ParseTimeoutFromFlag(timeoutStr, 5*time.Minute)

		opts := nelm.InstallOptions{
			BaseOptions: nelm.BaseOptions{
				ReleaseName: releaseName,
				Namespace:   namespace,
				KubeContext: kubeContext,
				AutoApprove: autoApprove,
				Timeout:     timeout,
			},
			Environment: env,
		}

		// Parse max concurrency
		maxConcurrency, _ := cmd.Flags().GetInt("max-concurrency")
		if maxConcurrency > 0 {
			opts.MaxConcurrency = maxConcurrency
		}

		// Usar função helper para logging
		nelm.LogNelmCommandStart("instalação", map[string]interface{}{
			"env":            env,
			"kubeContext":    kubeContext,
			"release":        releaseName,
			"namespace":      namespace,
			"autoApprove":    autoApprove,
			"timeout":        timeout,
			"maxConcurrency": opts.MaxConcurrency,
		})

		if err := nelm.ExecuteSmartInstall(&opts); err != nil {
			nelm.LogNelmCommandError("instalação", err)
			return err
		}
		nelm.LogNelmCommandSuccess("instalação")
		return nil
	},
}

func init() {
	// A flag 'release' agora é opcional.
	installCmd.Flags().StringP("release", "r", "", "Nome da release específica a ser instalada (opcional)")
	// A flag 'namespace' é opcional - se não fornecida, usa o nome da release como namespace.
	installCmd.Flags().StringP("namespace", "n", "", "Namespace onde a release será instalada (opcional - usa o nome da release se não fornecido)")

	installCmd.Flags().String("env", "", "O ambiente de destino (ex: stg, prd) (obrigatório)")
	installCmd.Flags().StringP("kube-context", "x", "", "Contexto do kubeconfig a ser usado (obrigatório)")
	installCmd.Flags().Bool("auto-approve", false, "Pula a confirmação interativa")
	installCmd.Flags().String("timeout", "5m", "Timeout para a operação (ex: 10s, 1m, 1h)")
	installCmd.Flags().Int("max-concurrency", 3, "Máximo de releases executadas em paralelo")

	installCmd.MarkFlagRequired("env")
	installCmd.MarkFlagRequired("kube-context")
}
