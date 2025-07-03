// cmd/argocd/repo/add.go
package repo

import (
	"context"
	"time"

	"github.com/estudosdevops/opsmaster/internal/argocd"
	"github.com/estudosdevops/opsmaster/internal/logger"
	"github.com/spf13/cobra"
)

var (
	repoUsername string
	repoPassword string
)

var repoAddCmd = &cobra.Command{
	Use:   "add <url-do-repositorio>",
	Short: "Adiciona um novo repositório Git ao Argo CD",
	Long:  `Registra um novo repositório Git no Argo CD. Se o repositório for privado, forneça as credenciais com as flags --username e --password.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		log := logger.Get()
		repoURL := args[0]
		serverAddr, _ := cmd.Flags().GetString("server")
		authToken, _ := cmd.Flags().GetString("token")
		insecure, _ := cmd.Flags().GetBool("insecure")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		log.Info("Adicionando repositório no Argo CD...", "repositório", repoURL)

		err := argocd.AddRepository(ctx, serverAddr, authToken, insecure, repoURL, repoUsername, repoPassword)
		if err != nil {
			log.Error("Falha ao adicionar o repositório", "erro", err)
			return err
		}

		log.Info("Repositório adicionado com sucesso!", "repositório", repoURL)
		return nil
	},
}

func init() {
	RepoCmd.AddCommand(repoAddCmd)
	repoAddCmd.Flags().StringVar(&repoUsername, "username", "", "Nome de usuário para repositórios privados")
	repoAddCmd.Flags().StringVar(&repoPassword, "password", "", "Senha ou token de acesso pessoal para repositórios privados")
}
