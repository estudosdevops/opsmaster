// cmd/argocd/argocd.go
package argocd

import (
	"fmt"

	"github.com/estudosdevops/opsmaster/cmd/argocd/app"
	"github.com/estudosdevops/opsmaster/cmd/argocd/project"
	"github.com/estudosdevops/opsmaster/cmd/argocd/repo"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	serverAddr string
	authToken  string
	insecure   bool
)

// ArgocdCmd é o comando pai "argocd".
var ArgocdCmd = &cobra.Command{
	Use:   "argocd",
	Short: "Gerencia interações com o Argo CD",
	Long:  `Um conjunto de comandos para interagir com a API do Argo CD.`,
	// PersistentPreRunE carrega e valida a configuração antes de qualquer subcomando ser executado.
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Se o usuário passou as flags de conexão, elas têm prioridade máxima.
		if cmd.Flag("server").Changed && cmd.Flag("token").Changed {
			// As variáveis globais (serverAddr, authToken, insecure) já são preenchidas
			// automaticamente pelo Cobra, então não precisamos fazer nada aqui.
			return nil
		}

		// Se as flags não foram passadas, tentamos usar o arquivo de configuração.
		contextName, _ := cmd.Flags().GetString("context")
		if contextName == "" {
			contextName = viper.GetString("current-context")
		}
		if contextName == "" {
			return fmt.Errorf("nenhum contexto definido e as flags --server e --token não foram fornecidas. Use a flag --context ou defina 'current-context' no seu ~/.opsmaster.yaml")
		}

		// Constrói as chaves para buscar no arquivo de configuração.
		serverKey := fmt.Sprintf("contexts.%s.argocd.server", contextName)
		tokenKey := fmt.Sprintf("contexts.%s.argocd.token", contextName)
		insecureKey := fmt.Sprintf("contexts.%s.argocd.insecure", contextName)

		// Preenche as variáveis globais com os valores do arquivo.
		serverAddr = viper.GetString(serverKey)
		authToken = viper.GetString(tokenKey)
		insecure = viper.GetBool(insecureKey)

		// Validação final: se, mesmo após ler o config, ainda não tivermos os valores, retorna um erro.
		if serverAddr == "" || authToken == "" {
			return fmt.Errorf("o endereço do servidor e o token do Argo CD são obrigatórios. Forneça-os via flags ou no arquivo de configuração para o contexto '%s'", contextName)
		}

		return nil
	},
}

func init() {
	// Adiciona os subcomandos 'app', 'project', e 'repo' ao comando pai 'argocd'.
	ArgocdCmd.AddCommand(app.AppCmd)
	ArgocdCmd.AddCommand(project.ProjectCmd)
	ArgocdCmd.AddCommand(repo.RepoCmd)

	// Define as flags persistentes que serão herdadas por todos os subcomandos de 'argocd'.
	ArgocdCmd.PersistentFlags().StringVar(&serverAddr, "server", "", "Endereço do servidor Argo CD (sobrescreve o config)")
	ArgocdCmd.PersistentFlags().StringVar(&authToken, "token", "", "Token de autenticação para a API do Argo CD (sobrescreve o config)")
	ArgocdCmd.PersistentFlags().BoolVar(&insecure, "insecure", false, "Pula a verificação de certificado TLS (sobrescreve o config)")
}
