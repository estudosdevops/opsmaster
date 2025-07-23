// cmd/argocd/cluster/list.go
package cluster

import (
	"context"
	"time"

	"github.com/estudosdevops/opsmaster/internal/argocd"
	"github.com/estudosdevops/opsmaster/internal/logger"
	"github.com/estudosdevops/opsmaster/internal/presenter"
	"github.com/spf13/cobra"
)

// clusterListCmd representa o comando "opsmaster argocd cluster list".
var clusterListCmd = &cobra.Command{
	Use:   "list",
	Short: "Lista todos os clusters registrados no Argo CD",
	Long:  `Busca e exibe todos os clusters Kubernetes registrados no Argo CD em um formato de tabela.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		log := logger.Get()
		serverAddr, _ := cmd.Flags().GetString("server")
		authToken, _ := cmd.Flags().GetString("token")
		insecure, _ := cmd.Flags().GetBool("insecure")

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		log.Info("Buscando lista de clusters no Argo CD...")

		clusterList, err := argocd.ListClusters(ctx, serverAddr, authToken, insecure)
		if err != nil {
			log.Error("Falha ao buscar os clusters", "erro", err)
			return err
		}

		if len(clusterList.Items) == 0 {
			log.Info("Nenhum cluster encontrado no Argo CD.")
			return nil
		}

		header := []string{"SERVIDOR", "NOME", "STATUS DA CONEXÃO", "VERSÃO DO SERVIDOR"}
		var rows [][]string
		for _, c := range clusterList.Items {
			rows = append(rows, []string{
				c.Server,
				c.Name,
				c.ConnectionState.Status,
				c.ServerVersion,
			})
		}

		presenter.PrintTable(header, rows)
		return nil
	},
}
