// opsmaster/internal/argocd/application.go
package argocd

import (
	"context"
	"fmt"

	"github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AppOptions contém todos os parâmetros necessários para criar uma aplicação.
type AppOptions struct {
	AppName        string
	Project        string
	DestinationNS  string
	RepoURL        string
	RepoPath       string
	TargetRevision string
	ValuesFile     string
	ImageRepo      string
	ImageTag       string
	DependencyName string
}

// CreateApplication constrói um objeto Application programaticamente e o envia para a API do Argo CD.
func CreateApplication(ctx context.Context, serverAddr, authToken string, insecure bool, opts AppOptions) error {
	// 1. Usa nosso helper para obter o cliente principal.
	apiClient, err := NewClient(serverAddr, authToken, insecure)
	if err != nil {
		return err
	}

	// 2. Obtém o cliente de serviço específico para "aplicações" e seu closer.
	appServiceCloser, appServiceClient, err := apiClient.NewApplicationClient()
	if err != nil {
		return fmt.Errorf("falha ao obter o cliente de aplicação: %w", err)
	}
	defer appServiceCloser.Close()

	// Constrói os caminhos completos para os parâmetros da imagem.
	imageRepoParam := fmt.Sprintf("%s.pods.image.name", opts.DependencyName)
	imageTagParam := fmt.Sprintf("%s.pods.image.tag", opts.DependencyName)

	// 3. Constrói o objeto Application programaticamente em Go.
	app := &v1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      opts.AppName,
			Namespace: "argocd",
		},
		Spec: v1alpha1.ApplicationSpec{
			Project: opts.Project,
			Source: &v1alpha1.ApplicationSource{
				RepoURL:        opts.RepoURL,
				Path:           opts.RepoPath,
				TargetRevision: opts.TargetRevision,
				Helm: &v1alpha1.ApplicationSourceHelm{
					ValueFiles: []string{opts.ValuesFile},
					Parameters: []v1alpha1.HelmParameter{
						{Name: imageRepoParam, Value: opts.ImageRepo},
						{Name: imageTagParam, Value: opts.ImageTag},
					},
				},
			},
			Destination: v1alpha1.ApplicationDestination{
				Server:    "https://kubernetes.default.svc",
				Namespace: opts.DestinationNS,
			},
			SyncPolicy: &v1alpha1.SyncPolicy{
				Automated: &v1alpha1.SyncPolicyAutomated{
					Prune:    true,
					SelfHeal: true,
				},
				SyncOptions: v1alpha1.SyncOptions{"CreateNamespace=true"},
			},
		},
	}

	// 4. Cria a requisição para a API.
	upsert := true
	createRequest := application.ApplicationCreateRequest{
		Application: app,
		Upsert:      &upsert,
	}

	// 5. Envia a requisição para criar ou atualizar a aplicação.
	_, err = appServiceClient.Create(ctx, &createRequest)
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return fmt.Errorf("a aplicação '%s' já existe no Argo CD", opts.AppName)
		}
		return fmt.Errorf("falha ao criar ou atualizar a aplicação no Argo CD: %w", err)
	}

	return nil
}

// DeleteApplication apaga uma aplicação específica no Argo CD.
func DeleteApplication(ctx context.Context, serverAddr, authToken string, insecure bool, appName string) error {
	// 1. Usa nosso helper para obter o cliente principal.
	apiClient, err := NewClient(serverAddr, authToken, insecure)
	if err != nil {
		return err
	}

	// 2. Obtém o cliente de serviço específico para "aplicações" e seu closer.
	appServiceCloser, appServiceClient, err := apiClient.NewApplicationClient()
	if err != nil {
		return fmt.Errorf("falha ao obter o cliente de aplicação: %w", err)
	}
	defer appServiceCloser.Close()

	// 3. Cria a requisição para deletar a aplicação.
	deleteRequest := application.ApplicationDeleteRequest{
		Name: &appName,
	}
	_, err = appServiceClient.Delete(ctx, &deleteRequest)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil
		}
		return fmt.Errorf("falha ao deletar a aplicação '%s': %w", appName, err)
	}

	return nil
}
