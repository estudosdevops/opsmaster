// cmd/argocd/project/create.go
package project

import (
	"context"
	"time"

	"github.com/estudosdevops/opsmaster/internal/argocd"
	"github.com/estudosdevops/opsmaster/internal/logger"
	"github.com/spf13/cobra"
)

var (
	projDescription string
	sourceRepos     []string
)

var projectCreateCmd = &cobra.Command{
	Use:   "create <nome-do-projeto>",
	Short: "Cria um novo projeto no Argo CD",
	Long:  `Cria um novo AppProject no Argo CD com uma descrição e repositórios de origem permitidos.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		log := logger.Get()
		projectName := args[0]
		serverAddr, _ := cmd.Flags().GetString("server")
		authToken, _ := cmd.Flags().GetString("token")
		insecure, _ := cmd.Flags().GetBool("insecure")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		log.Info("Criando projeto no Argo CD...",
			"projeto", projectName,
			"descrição", projDescription,
			"repositorios", sourceRepos,
		)

		err := argocd.CreateProject(ctx, serverAddr, authToken, insecure, projectName, projDescription, sourceRepos)
		if err != nil {
			log.Error("Falha ao criar o projeto", "erro", err)
			return err
		}
		log.Info("Projeto criado com sucesso!", "projeto", projectName)
		return nil
	},
}

func init() {
	projectCreateCmd.Flags().StringVarP(&projDescription, "description", "d", "", "Descrição do projeto")
	projectCreateCmd.Flags().StringSliceVarP(&sourceRepos, "source-repo", "s", []string{"*"}, "Repositório Git permitido para este projeto")
}
