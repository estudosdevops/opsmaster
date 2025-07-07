package project

import (
	"context"
	"time"

	"github.com/estudosdevops/opsmaster/internal/argocd"
	"github.com/estudosdevops/opsmaster/internal/logger"
	"github.com/spf13/cobra"
)

var projectDeleteCmd = &cobra.Command{
	Use:   "delete <nome-do-projeto>",
	Short: "Apaga um projeto do Argo CD",
	Long:  `Apaga um AppProject específico do Argo CD. Esta ação não pode ser desfeita.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		log := logger.Get()
		projectName := args[0]
		serverAddr, _ := cmd.Flags().GetString("server")
		authToken, _ := cmd.Flags().GetString("token")
		insecure, _ := cmd.Flags().GetBool("insecure")

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		log.Info("Apagando o projeto do Argo CD...", "projeto", projectName)
		if err := argocd.DeleteProject(ctx, serverAddr, authToken, insecure, projectName); err != nil {
			log.Error("Falha ao apagar o projeto", "erro", err)
			return err
		}
		log.Info("Projeto apagado com sucesso!", "projeto", projectName)
		return nil
	},
}
