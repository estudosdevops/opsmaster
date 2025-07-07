package repo

import (
	"context"
	"time"

	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/estudosdevops/opsmaster/internal/argocd"
	"github.com/estudosdevops/opsmaster/internal/logger"
	"github.com/estudosdevops/opsmaster/internal/presenter"
	"github.com/spf13/cobra"
)

var repoListCmd = &cobra.Command{
	Use:   "list [url-do-repositorio]",
	Short: "Lista todos os repositórios ou um repositório específico",
	Long:  `Busca e exibe repositórios Git registrados no Argo CD. Se uma URL for fornecida, exibe apenas os detalhes daquele repositório.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		log := logger.Get()
		serverAddr, _ := cmd.Flags().GetString("server")
		authToken, _ := cmd.Flags().GetString("token")
		insecure, _ := cmd.Flags().GetBool("insecure")

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		var repos []v1alpha1.Repository
		var err error

		if len(args) > 0 {
			repoURL := args[0]
			log.Info("Buscando repositório específico...", "repositório", repoURL)
			var repo *v1alpha1.Repository
			repo, err = argocd.GetRepository(ctx, serverAddr, authToken, insecure, repoURL)
			if err != nil {
				log.Error("Falha ao buscar o repositório", "erro", err)
				return err
			}
			if repo != nil {
				repos = append(repos, *repo)
			}
		} else {
			log.Info("Buscando lista de todos os repositórios...")
			repos, err = argocd.ListRepositories(ctx, serverAddr, authToken, insecure)
			if err != nil {
				log.Error("Falha ao buscar os repositórios", "erro", err)
				return err
			}
		}

		if len(repos) == 0 {
			log.Info("Nenhum repositório encontrado.")
			return nil
		}

		header := []string{"URL DO REPOSITÓRIO", "NOME DE USUÁRIO"}
		var rows [][]string
		for _, r := range repos {
			rows = append(rows, []string{r.Repo, r.Username})
		}
		presenter.PrintTable(header, rows)
		return nil
	},
}
