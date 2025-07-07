// cmd/argocd/app/list.go
package app

import (
	"context"
	"time"

	"github.com/estudosdevops/opsmaster/internal/argocd"
	"github.com/estudosdevops/opsmaster/internal/logger"
	"github.com/estudosdevops/opsmaster/internal/presenter"

	"github.com/spf13/cobra"
)

// appListCmd representa o comando "opsmaster argocd app list".
var appListCmd = &cobra.Command{
	Use:   "list [nome-da-aplicacao]",
	Short: "Lista todas as aplicações ou uma aplicação específica no Argo CD",
	Long:  `Busca e exibe aplicações gerenciadas pelo Argo CD. Se um nome de aplicação for fornecido, exibe apenas os detalhes daquela aplicação.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		log := logger.Get()
		serverAddr, _ := cmd.Flags().GetString("server")
		authToken, _ := cmd.Flags().GetString("token")
		insecure, _ := cmd.Flags().GetBool("insecure")

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		var apps []argocd.AppStatusInfo
		var err error

		if len(args) > 0 {
			appName := args[0]
			log.Info("Buscando aplicação específica...", "aplicação", appName)
			var app *argocd.AppStatusInfo
			app, err = argocd.GetApplication(ctx, serverAddr, authToken, insecure, appName)
			if err == nil && app != nil {
				apps = append(apps, *app)
			}
		} else {
			log.Info("Buscando lista de todas as aplicações no Argo CD...")
			apps, err = argocd.ListApplications(ctx, serverAddr, authToken, insecure)
		}

		if err != nil {
			log.Error("Falha ao buscar dados do Argo CD", "erro", err)
			return err
		}

		if len(apps) == 0 {
			log.Info("Nenhuma aplicação encontrada.")
			return nil
		}

		// Prepara os dados para a tabela.
		header := []string{"NOME", "PROJETO", "STATUS SYNC", "STATUS SAÚDE", "REPOSITÓRIO"}
		var rows [][]string
		for _, app := range apps {
			row := []string{
				app.Name,
				app.Project,
				string(app.SyncStatus),
				string(app.HealthStatus.Status),
				app.RepoURL,
			}
			rows = append(rows, row)
		}

		presenter.PrintTable(header, rows)

		return nil
	},
}
