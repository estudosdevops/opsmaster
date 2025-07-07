package project

import (
	"context"
	"time"

	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/estudosdevops/opsmaster/internal/argocd"
	"github.com/estudosdevops/opsmaster/internal/logger"
	"github.com/estudosdevops/opsmaster/internal/presenter"
	"github.com/spf13/cobra"
)

var projectListCmd = &cobra.Command{
	Use:   "list [nome-do-projeto]",
	Short: "Lista todos os projetos ou um projeto específico",
	Long:  `Busca e exibe projetos registrados no Argo CD. Se um nome de projeto for fornecido, exibe apenas os detalhes daquele projeto.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		log := logger.Get()
		serverAddr, _ := cmd.Flags().GetString("server")
		authToken, _ := cmd.Flags().GetString("token")
		insecure, _ := cmd.Flags().GetBool("insecure")

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		var projects []v1alpha1.AppProject
		var err error

		if len(args) > 0 {
			projectName := args[0]
			log.Info("Buscando projeto específico...", "projeto", projectName)
			var project *v1alpha1.AppProject
			project, err = argocd.GetProject(ctx, serverAddr, authToken, insecure, projectName)
			// CORREÇÃO: Trata o erro imediatamente. Se não houver erro, adiciona o projeto à lista.
			if err != nil {
				log.Error("Falha ao buscar o projeto", "erro", err)
				return err
			}
			if project != nil {
				projects = append(projects, *project)
			}
		} else {
			log.Info("Buscando lista de todos os projetos...")
			projects, err = argocd.ListProjects(ctx, serverAddr, authToken, insecure)
			if err != nil {
				log.Error("Falha ao buscar os projetos", "erro", err)
				return err
			}
		}

		if len(projects) == 0 {
			log.Info("Nenhum projeto encontrado.")
			return nil
		}

		header := []string{"NOME", "DESCRIÇÃO"}
		var rows [][]string
		for _, p := range projects {
			rows = append(rows, []string{p.Name, p.Spec.Description})
		}
		presenter.PrintTable(header, rows)
		return nil
	},
}
