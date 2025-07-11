// opsmaster/internal/argocd/status.go
package argocd

import (
	"context"
	"fmt"
	"time"

	"github.com/estudosdevops/opsmaster/internal/logger"

	"github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient/project"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient/repository"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
)

// AppStatusInfo agrupa as informações de status de uma aplicação.
type AppStatusInfo struct {
	Name         string
	Project      string
	SyncStatus   v1alpha1.SyncStatusCode
	HealthStatus v1alpha1.HealthStatus
	RepoURL      string
}

// --- Funções de Aplicação ---

// GetApplicationDetails busca o objeto completo de uma aplicação.
// Esta é agora a nossa função "base" para buscar uma aplicação.
func GetApplicationDetails(ctx context.Context, serverAddr, authToken string, insecure bool, appName string) (*v1alpha1.Application, error) {
	apiClient, err := NewClient(serverAddr, authToken, insecure)
	if err != nil {
		return nil, err
	}

	appServiceCloser, appServiceClient, err := apiClient.NewApplicationClient()
	if err != nil {
		return nil, fmt.Errorf("falha ao obter o cliente de aplicação: %w", err)
	}
	defer appServiceCloser.Close()

	// A função Get da API já retorna o objeto completo da aplicação.
	app, err := appServiceClient.Get(ctx, &application.ApplicationQuery{Name: &appName})
	if err != nil {
		return nil, fmt.Errorf("falha ao buscar a aplicação '%s': %w", appName, err)
	}

	return app, nil
}

// GetApplication busca e transforma o estado de uma aplicação para um formato simplificado.
func GetApplication(ctx context.Context, serverAddr, authToken string, insecure bool, appName string) (*AppStatusInfo, error) {
	// Reutiliza a nossa função base para buscar os detalhes completos.
	app, err := GetApplicationDetails(ctx, serverAddr, authToken, insecure, appName)
	if err != nil {
		return nil, err
	}

	// Transforma o objeto completo na nossa struct simplificada.
	return &AppStatusInfo{
		Name:         app.Name,
		Project:      app.Spec.Project,
		SyncStatus:   app.Status.Sync.Status,
		HealthStatus: app.Status.Health,
		RepoURL:      app.Spec.Source.RepoURL,
	}, nil
}

// ListApplications busca todas as aplicações no Argo CD.
func ListApplications(ctx context.Context, serverAddr, authToken string, insecure bool) ([]AppStatusInfo, error) {
	apiClient, err := NewClient(serverAddr, authToken, insecure)
	if err != nil {
		return nil, err
	}

	appServiceCloser, appServiceClient, err := apiClient.NewApplicationClient()
	if err != nil {
		return nil, fmt.Errorf("falha ao obter o cliente de aplicação: %w", err)
	}
	defer appServiceCloser.Close()

	appList, err := appServiceClient.List(ctx, &application.ApplicationQuery{})
	if err != nil {
		return nil, fmt.Errorf("falha ao listar as aplicações: %w", err)
	}

	var statuses []AppStatusInfo
	for _, app := range appList.Items {
		statuses = append(statuses, AppStatusInfo{
			Name:         app.Name,
			Project:      app.Spec.Project,
			SyncStatus:   app.Status.Sync.Status,
			HealthStatus: app.Status.Health,
			RepoURL:      app.Spec.Source.RepoURL, // Coleta a URL do repositório.
		})
	}

	return statuses, nil
}

// WaitForAppStatus espera que uma aplicação atinja o estado Healthy e Synced.
func WaitForAppStatus(ctx context.Context, serverAddr, authToken string, insecure bool, appName string, interval time.Duration) (*v1alpha1.Application, error) {
	log := logger.Get()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Info("Aguardando a aplicação ficar saudável e sincronizada...", "aplicação", appName)

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("tempo esgotado aguardando a aplicação '%s': %w", appName, ctx.Err())
		case <-ticker.C:
			// Busca os detalhes completos da aplicação a cada verificação.
			app, err := GetApplicationDetails(ctx, serverAddr, authToken, insecure, appName)
			if err != nil {
				log.Error("Erro ao buscar o status da aplicação, tentando novamente...", "erro", err)
				continue
			}

			healthStatus := app.Status.Health.Status
			syncStatus := app.Status.Sync.Status

			log.Info("Verificando status...", "saúde", healthStatus, "sincronização", syncStatus)

			if string(healthStatus) == "Healthy" && string(syncStatus) == "Synced" {
				return app, nil // Sucesso! Retorna o objeto completo da aplicação.
			}
		}
	}
}

// --- Funções de Projeto ---

// GetProject busca um projeto específico pelo nome.
func GetProject(ctx context.Context, serverAddr, authToken string, insecure bool, projName string) (*v1alpha1.AppProject, error) {
	apiClient, err := NewClient(serverAddr, authToken, insecure)
	if err != nil {
		return nil, err
	}
	projServiceCloser, projServiceClient, err := apiClient.NewProjectClient()
	if err != nil {
		return nil, fmt.Errorf("falha ao obter o cliente de projeto: %w", err)
	}
	defer projServiceCloser.Close()

	p, err := projServiceClient.Get(ctx, &project.ProjectQuery{Name: projName})
	if err != nil {
		return nil, fmt.Errorf("falha ao buscar o projeto '%s': %w", projName, err)
	}
	return p, nil
}

// ListProjects busca todos os projetos registrados no Argo CD.
func ListProjects(ctx context.Context, serverAddr, authToken string, insecure bool) ([]v1alpha1.AppProject, error) {
	apiClient, err := NewClient(serverAddr, authToken, insecure)
	if err != nil {
		return nil, err
	}
	projServiceCloser, projServiceClient, err := apiClient.NewProjectClient()
	if err != nil {
		return nil, fmt.Errorf("falha ao obter o cliente de projeto: %w", err)
	}
	defer projServiceCloser.Close()

	projectList, err := projServiceClient.List(ctx, &project.ProjectQuery{})
	if err != nil {
		return nil, fmt.Errorf("falha ao listar os projetos: %w", err)
	}
	return projectList.Items, nil
}

// --- Funções de Repositório ---
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
