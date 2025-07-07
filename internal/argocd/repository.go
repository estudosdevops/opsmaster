// opsmaster/internal/argocd/repository.go
package argocd

import (
	"context"
	"fmt"

	"github.com/argoproj/argo-cd/v2/pkg/apiclient/repository"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AddRepository registra um novo repositório Git no Argo CD.
// Se username e password estiverem vazios, registra como um repositório público.
func AddRepository(ctx context.Context, serverAddr, authToken string, insecure bool, repoURL, username, password string) error {
	// 1. Usa o nosso helper para obter o cliente principal.
	apiClient, err := NewClient(serverAddr, authToken, insecure)
	if err != nil {
		return err
	}

	// 2. Obtém o cliente de serviço específico para "repositórios" e o seu closer.
	repoServiceCloser, repoServiceClient, err := apiClient.NewRepoClient()
	if err != nil {
		return fmt.Errorf("falha ao obter o cliente de repositório: %w", err)
	}
	defer repoServiceCloser.Close()

	// 3. Monta o objeto do repositório.
	repo := &v1alpha1.Repository{
		Repo: repoURL,
	}

	// 4. Se credenciais foram fornecidas, adiciona ao objeto.
	if username != "" && password != "" {
		repo.Username = username
		repo.Password = password
	}

	// 5. Cria a requisição para a API.
	createRequest := &repository.RepoCreateRequest{
		Repo:   repo,
		Upsert: true, // Upsert = true significa que se o repo já existir, ele será atualizado.
	}

	// 6. Envia a requisição para criar/registrar o repositório.
	_, err = repoServiceClient.Create(ctx, createRequest)
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return fmt.Errorf("o repositório '%s' já está registrado no Argo CD", repoURL)
		}
		return fmt.Errorf("falha ao adicionar o repositório no Argo CD: %w", err)
	}

	return nil
}

// DeleteRepository remove o registro de um repositório Git no Argo CD.
func DeleteRepository(ctx context.Context, serverAddr, authToken string, insecure bool, repoURL string) error {
	apiClient, err := NewClient(serverAddr, authToken, insecure)
	if err != nil {
		return err
	}
	repoServiceCloser, repoServiceClient, err := apiClient.NewRepoClient()
	if err != nil {
		return fmt.Errorf("falha ao obter o cliente de repositório: %w", err)
	}
	defer repoServiceCloser.Close()

	_, err = repoServiceClient.DeleteRepository(ctx, &repository.RepoQuery{Repo: repoURL})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil
		}
		return fmt.Errorf("falha ao apagar o repositório '%s': %w", repoURL, err)
	}
	return nil
}

// GetRepository busca um repositório específico pela URL.
func GetRepository(ctx context.Context, serverAddr, authToken string, insecure bool, repoURL string) (*v1alpha1.Repository, error) {
	apiClient, err := NewClient(serverAddr, authToken, insecure)
	if err != nil {
		return nil, err
	}
	repoServiceCloser, repoServiceClient, err := apiClient.NewRepoClient()
	if err != nil {
		return nil, fmt.Errorf("falha ao obter o cliente de repositório: %w", err)
	}
	defer repoServiceCloser.Close()

	repo, err := repoServiceClient.Get(ctx, &repository.RepoQuery{Repo: repoURL})
	if err != nil {
		return nil, fmt.Errorf("falha ao buscar o repositório '%s': %w", repoURL, err)
	}
	return repo, nil
}

// ListRepositories busca todos os repositórios registrados no Argo CD.
func ListRepositories(ctx context.Context, serverAddr, authToken string, insecure bool) ([]v1alpha1.Repository, error) {
	apiClient, err := NewClient(serverAddr, authToken, insecure)
	if err != nil {
		return nil, err
	}
	repoServiceCloser, repoServiceClient, err := apiClient.NewRepoClient()
	if err != nil {
		return nil, fmt.Errorf("falha ao obter o cliente de repositório: %w", err)
	}
	defer repoServiceCloser.Close()

	repoList, err := repoServiceClient.List(ctx, &repository.RepoQuery{})
	if err != nil {
		return nil, fmt.Errorf("falha ao listar os repositórios: %w", err)
	}

	repos := make([]v1alpha1.Repository, len(repoList.Items))
	for i, item := range repoList.Items {
		repos[i] = *item
	}
	return repos, nil
}
