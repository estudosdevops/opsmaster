// cmd/argocd/app/get.go
package app

import (
	"context"
	"fmt"
	"time"

	"github.com/estudosdevops/opsmaster/internal/argocd"
	"github.com/estudosdevops/opsmaster/internal/logger"
	"github.com/estudosdevops/opsmaster/internal/presenter" // Importa o nosso pacote de tabelas
	"github.com/spf13/cobra"
)

// appGetCmd representa o comando "opsmaster argocd app get".
var appGetCmd = &cobra.Command{
	Use:   "get <nome-da-aplicacao>",
	Short: "Exibe detalhes de uma aplicação específica",
	Long:  `Busca e exibe informações detalhadas sobre uma única aplicação no Argo CD, incluindo status, repositório e recursos sincronizados.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		log := logger.Get()
		appName := args[0]
		serverAddr, _ := cmd.Flags().GetString("server")
		authToken, _ := cmd.Flags().GetString("token")
		insecure, _ := cmd.Flags().GetBool("insecure")

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		log.Info("Buscando detalhes da aplicação...", "aplicação", appName)

		// Chama a nossa lógica para buscar os detalhes completos.
		app, err := argocd.GetApplicationDetails(ctx, serverAddr, authToken, insecure, appName)
		if err != nil {
			log.Error("Falha ao buscar os detalhes da aplicação", "erro", err)
			return err
		}

		// Tabela 1: Informações Gerais e Status
		detailsHeader := []string{"CAMPO", "VALOR"}
		var detailsRows [][]string
		detailsRows = append(detailsRows, []string{"Projeto", app.Spec.Project})
		detailsRows = append(detailsRows, []string{"Namespace", app.Spec.Destination.Namespace})
		detailsRows = append(detailsRows, []string{"Cluster", app.Spec.Destination.Server})
		detailsRows = append(detailsRows, []string{"Status de Saúde", string(app.Status.Health.Status)})
		detailsRows = append(detailsRows, []string{"Status de Sincronização", string(app.Status.Sync.Status)})
		if app.Status.Health.Message != "" {
			detailsRows = append(detailsRows, []string{"Mensagem de Saúde", app.Status.Health.Message})
		}
		detailsRows = append(detailsRows, []string{"Repositório", app.Spec.Source.RepoURL})
		detailsRows = append(detailsRows, []string{"Revisão de Destino", app.Spec.Source.TargetRevision})
		detailsRows = append(detailsRows, []string{"Caminho no Repo", app.Spec.Source.Path})
		presenter.PrintTable(detailsHeader, detailsRows)

		// Adiciona uma linha em branco para separar as tabelas.
		fmt.Println()

		// Tabela 2: Recursos Sincronizados
		if len(app.Status.Resources) > 0 {
			resourcesHeader := []string{"GRUPO", "TIPO", "NAMESPACE", "NOME", "STATUS", "SAÚDE", "MENSAGEM"}
			var resourcesRows [][]string
			for _, res := range app.Status.Resources {
				healthMsg := ""
				// CORREÇÃO: Lógica para exibir "OK" quando o recurso está saudável.
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
					healthMsg, // Adiciona a mensagem de saúde à linha.
				})
			}
			presenter.PrintTable(resourcesHeader, resourcesRows)
		} else {
			fmt.Println("  Nenhum recurso encontrado.")
		}

		return nil
	},
}
