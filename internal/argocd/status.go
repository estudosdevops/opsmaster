// opsmaster/internal/argocd/status.go
package argocd

import (
	"context"
	"fmt"
	"time"

	"github.com/estudosdevops/opsmaster/internal/logger"

	"github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
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

// GetApplication busca o estado atual de uma aplicação específica.
func GetApplication(ctx context.Context, serverAddr, authToken string, insecure bool, appName string) (*AppStatusInfo, error) {
	apiClient, err := NewClient(serverAddr, authToken, insecure)
	if err != nil {
		return nil, err
	}

	appServiceCloser, appServiceClient, err := apiClient.NewApplicationClient()
	if err != nil {
		return nil, fmt.Errorf("falha ao obter o cliente de aplicação: %w", err)
	}
	defer appServiceCloser.Close()

	app, err := appServiceClient.Get(ctx, &application.ApplicationQuery{Name: &appName})
	if err != nil {
		return nil, fmt.Errorf("falha ao buscar a aplicação '%s': %w", appName, err)
	}

	return &AppStatusInfo{
		Name:         app.Name,
		Project:      app.Spec.Project,
		SyncStatus:   app.Status.Sync.Status,
		HealthStatus: app.Status.Health,
		RepoURL:      app.Spec.Source.RepoURL, // Coleta a URL do repositório.
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
func WaitForAppStatus(ctx context.Context, serverAddr, authToken string, insecure bool, appName string, interval time.Duration) error {
	log := logger.Get()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Info("Aguardando a aplicação ficar saudável e sincronizada...", "aplicação", appName)

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("tempo esgotado aguardando a aplicação '%s': %w", appName, ctx.Err())
		case <-ticker.C:
			app, err := GetApplication(ctx, serverAddr, authToken, insecure, appName)
			if err != nil {
				log.Error("Erro ao buscar o status da aplicação, tentando novamente...", "erro", err)
				continue
			}

			log.Info("Verificando status...", "saúde", app.HealthStatus.Status, "sincronização", app.SyncStatus)

			if string(app.HealthStatus.Status) == "Healthy" && string(app.SyncStatus) == "Synced" {
				return nil
			}
		}
	}
}
