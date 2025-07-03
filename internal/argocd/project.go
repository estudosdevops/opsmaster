// opsmaster/internal/argocd/project.go
package argocd

import (
	"context"
	"fmt"

	"github.com/argoproj/argo-cd/v2/pkg/apiclient/project"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateProject conecta-se à API do Argo CD e cria um novo projeto.
func CreateProject(ctx context.Context, serverAddr, authToken string, insecure bool, projName, description string, sourceRepos []string) error {
	// Usa nosso helper para obter o cliente principal.
	apiClient, err := NewClient(serverAddr, authToken, insecure)
	if err != nil {
		return err
	}

	// Obtém o cliente de serviço específico para "projetos" e seu closer.
	projServiceCloser, projServiceClient, err := apiClient.NewProjectClient()
	if err != nil {
		return fmt.Errorf("falha ao obter o cliente de projeto: %w", err)
	}
	defer projServiceCloser.Close()

	projSpec := v1alpha1.AppProjectSpec{
		Description: description,
		SourceRepos: sourceRepos,
		Destinations: []v1alpha1.ApplicationDestination{
			{Server: "*", Namespace: "*"},
		},
		ClusterResourceWhitelist: []metav1.GroupKind{{Group: "*", Kind: "*"}},
	}

	argoProject := &v1alpha1.AppProject{
		ObjectMeta: metav1.ObjectMeta{Name: projName},
		Spec:       projSpec,
	}

	createRequest := project.ProjectCreateRequest{
		Project: argoProject,
		Upsert:  true, // Upsert = true significa que se o projeto já existir, ele será atualizado.
	}

	_, err = projServiceClient.Create(ctx, &createRequest)
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return fmt.Errorf("o projeto '%s' já existe no Argo CD", projName)
		}
		return fmt.Errorf("falha ao criar o projeto no Argo CD: %w", err)
	}

	return nil
}
