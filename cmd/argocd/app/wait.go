// cmd/argocd/app/wait.go
package app

import (
	"context"
	"fmt"
	"time"

	"github.com/estudosdevops/opsmaster/internal/argocd"
	"github.com/estudosdevops/opsmaster/internal/logger"
	"github.com/estudosdevops/opsmaster/internal/presenter"
	"github.com/spf13/cobra"
)

var (
	waitTimeout     time.Duration
	waitInterval    time.Duration
	waitShowDetails bool // Variável para a nova flag
)

// appWaitCmd representa o comando "opsmaster argocd app wait".
var appWaitCmd = &cobra.Command{
	Use:   "wait <nome-da-aplicacao>",
	Short: "Aguarda uma aplicação ficar saudável e sincronizada",
	Long: `Monitora continuamente uma aplicação no Argo CD e encerra com sucesso quando
o status de saúde for 'Healthy' e o status de sincronização for 'Synced'.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		log := logger.Get()
		appName := args[0]
		serverAddr, _ := cmd.Flags().GetString("server")
		authToken, _ := cmd.Flags().GetString("token")
		insecure, _ := cmd.Flags().GetBool("insecure")

		ctx, cancel := context.WithTimeout(context.Background(), waitTimeout)
		defer cancel()

		// A função de espera agora retorna o objeto completo da aplicação.
		finalApp, err := argocd.WaitForAppStatus(ctx, serverAddr, authToken, insecure, appName, waitInterval)
		if err != nil {
			log.Error("A aplicação não atingiu o estado desejado a tempo", "erro", err)
			return err
		}

		log.Info("Aplicação está saudável e sincronizada!", "aplicação", appName)

		// Se a flag --show-details foi passada, exibe os detalhes.
		if waitShowDetails {
			fmt.Println() // Adiciona uma linha em branco para separar.

			// Tabela 1: Detalhes da Aplicação (idêntica ao comando 'get')
			detailsHeader := []string{"CAMPO", "VALOR"}
			var detailsRows [][]string
			detailsRows = append(detailsRows, []string{"Projeto", finalApp.Spec.Project})
			detailsRows = append(detailsRows, []string{"Namespace de Destino", finalApp.Spec.Destination.Namespace})
			detailsRows = append(detailsRows, []string{"Cluster de Destino", finalApp.Spec.Destination.Server})
			detailsRows = append(detailsRows, []string{"Status de Saúde", string(finalApp.Status.Health.Status)})
			detailsRows = append(detailsRows, []string{"Status de Sincronização", string(finalApp.Status.Sync.Status)})
			if finalApp.Status.Health.Message != "" {
				detailsRows = append(detailsRows, []string{"Mensagem de Saúde", finalApp.Status.Health.Message})
			}
			detailsRows = append(detailsRows, []string{"Repositório de Origem", finalApp.Spec.Source.RepoURL})
			detailsRows = append(detailsRows, []string{"Revisão de Destino", finalApp.Spec.Source.TargetRevision})
			detailsRows = append(detailsRows, []string{"Caminho no Repositório", finalApp.Spec.Source.Path})
			presenter.PrintTable(detailsHeader, detailsRows)

			// Adiciona uma linha em branco para separar as tabelas.
			fmt.Println()

			// Tabela 2: Recursos Sincronizados (idêntica ao comando 'get')
			if len(finalApp.Status.Resources) > 0 {
				resourcesHeader := []string{"GRUPO", "TIPO", "NAMESPACE", "NOME", "STATUS", "SAÚDE", "MENSAGEM"}
				var resourcesRows [][]string
				for _, res := range finalApp.Status.Resources {
					healthMsg := ""
					if res.Health != nil {
						if res.Health.Status == "Healthy" && res.Health.Message == "" {
							healthMsg = "OK"
						} else {
							healthMsg = res.Health.Message
						}
					}
					resourcesRows = append(resourcesRows, []string{
						res.Group,
						res.Kind,
						res.Namespace,
						res.Name,
						string(res.Status),
						string(res.Health.Status),
						healthMsg,
					})
				}
				presenter.PrintTable(resourcesHeader, resourcesRows)
			} else {
				log.Info("Nenhum recurso sincronizado encontrado para esta aplicação.")
			}
		}

		return nil
	},
}

func init() {
	appWaitCmd.Flags().DurationVarP(&waitTimeout, "timeout", "t", 5*time.Minute, "Tempo máximo de espera pela aplicação")
	appWaitCmd.Flags().DurationVarP(&waitInterval, "interval", "i", 15*time.Second, "Intervalo entre as verificações de status")

	// Adiciona a nova flag opcional.
	appWaitCmd.Flags().BoolVar(&waitShowDetails, "show-details", false, "Exibe os detalhes da aplicação após a conclusão bem-sucedida")
}
