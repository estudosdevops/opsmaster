// opsmaster/internal/argocd/client.go
package argocd

import (
	"fmt"

	"github.com/argoproj/argo-cd/v2/pkg/apiclient"
)

// NewClient centraliza a criação e configuração do cliente da API do Argo CD.
// Esta é a nossa função "helper" que será reutilizada.
// A função agora retorna apenas o cliente principal e um erro.
func NewClient(serverAddr, authToken string, insecure bool) (apiclient.Client, error) {
	clientOptions := apiclient.ClientOptions{
		ServerAddr: serverAddr,
		AuthToken:  authToken,
		GRPCWeb:    true,
		Insecure:   insecure,
	}

	// A função NewClient da biblioteca do Argo CD retorna 2 valores: o cliente e um erro.
	apiClient, err := apiclient.NewClient(&clientOptions)
	if err != nil {
		return nil, fmt.Errorf("falha ao criar o cliente da API do Argo CD: %w", err)
	}

	return apiClient, nil
}
