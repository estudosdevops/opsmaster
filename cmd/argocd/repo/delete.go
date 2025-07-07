package repo

import (
	"context"
	"time"

	"github.com/estudosdevops/opsmaster/internal/argocd"
	"github.com/estudosdevops/opsmaster/internal/logger"
	"github.com/spf13/cobra"
)

var repoDeleteCmd = &cobra.Command{
	Use:   "delete <url-do-repositorio>",
	Short: "Remove um repositório do Argo CD",
	Long:  `Remove o registro de um repositório Git do Argo CD. Esta ação não apaga o repositório no Git.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		log := logger.Get()
		repoURL := args[0]
		serverAddr, _ := cmd.Flags().GetString("server")
		authToken, _ := cmd.Flags().GetString("token")
		insecure, _ := cmd.Flags().GetBool("insecure")

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		log.Info("Apagando o repositório do Argo CD...", "repositório", repoURL)
		if err := argocd.DeleteRepository(ctx, serverAddr, authToken, insecure, repoURL); err != nil {
			log.Error("Falha ao apagar o repositório", "erro", err)
			return err
		}
		log.Info("Repositório apagado com sucesso!", "repositório", repoURL)
		return nil
	},
}
