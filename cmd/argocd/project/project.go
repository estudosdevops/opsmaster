// cmd/argocd/project/project.go
package project

import (
	"github.com/spf13/cobra"
)

// ProjectCmd é o comando "project" dentro do grupo "argocd".
var ProjectCmd = &cobra.Command{
	Use:   "project",
	Short: "Gerencia projetos do Argo CD",
	Long:  `Um conjunto de subcomandos para criar e gerenciar projetos no Argo CD.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	ProjectCmd.AddCommand(projectCreateCmd)
}
