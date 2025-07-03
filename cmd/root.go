// cmd/root.go
package cmd

import (
	"github.com/estudosdevops/opsmaster/cmd/argocd"
	"github.com/estudosdevops/opsmaster/cmd/get"
	"github.com/estudosdevops/opsmaster/cmd/scan"

	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// RootCmd é o comando raiz da nossa aplicação.
var RootCmd = &cobra.Command{
	Use:   "opsmaster",
	Short: "OpsMaster - Uma ferramenta de CLI para operações de DevOps",
	Long: `OpsMaster é uma ferramenta de CLI projetada para ajudar em várias
operações de DevOps. Ela fornece um conjunto de comandos para automatizar e
simplificar tarefas comuns de DevOps.`,
}

func Execute() error {
	return RootCmd.Execute()
}

func init() {
	RootCmd.AddCommand(scan.ScanCmd)
	RootCmd.AddCommand(get.GetCmd)
	RootCmd.AddCommand(argocd.ArgocdCmd)

	cobra.OnInitialize(initConfig)
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "arquivo de configuração (o padrão é $HOME/.opsmaster.yaml)")
	RootCmd.PersistentFlags().String("context", "", "O contexto a ser usado do arquivo de configuração (ex: staging, producao)")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".opsmaster")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Usando o arquivo de configuração:", viper.ConfigFileUsed())
	}
}
