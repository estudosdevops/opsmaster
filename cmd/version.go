package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// versionCmd representa o comando "version".
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Exibe o número da versão do OpsMaster",
	Long:  `Exibe o número da versão da ferramenta de CLI OpsMaster.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("OpsMaster v0.1.0")
	},
}

func init() {
	// Adiciona o comando 'version' ao comando raiz 'rootCmd'.
	RootCmd.AddCommand(versionCmd)
}
