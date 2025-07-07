// cmd/argocd/repo/repo.go
package repo

import (
	"github.com/spf13/cobra"
)

// RepoCmd é o comando "repo" dentro do grupo "argocd".
var RepoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Gerencia repositórios do Argo CD",
	Long:  `Um conjunto de subcomandos para adicionar e gerenciar repositórios no Argo CD.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	RepoCmd.AddCommand(repoAddCmd)
	RepoCmd.AddCommand(repoDeleteCmd)
	RepoCmd.AddCommand(repoListCmd)
}
