// cmd/argocd/cluster/cluster.go
package cluster

import (
	"github.com/spf13/cobra"
)

// ClusterCmd representa o comando "cluster" dentro do grupo "argocd". É exportado.
var ClusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Gerencia clusters Kubernetes registrados no Argo CD",
	Long:  `Um conjunto de subcomandos para adicionar, listar e remover clusters do Argo CD.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	ClusterCmd.AddCommand(clusterListCmd)
}
