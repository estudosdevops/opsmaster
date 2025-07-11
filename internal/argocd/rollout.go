// internal/argocd/rollout.go
package argocd

import (
	"context"
	"fmt"

	"github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
)

// runRolloutAction é uma função helper interna que executa uma ação específica em um Rollout.
func runRolloutAction(ctx context.Context, serverAddr, authToken string, insecure bool, appName, action string) error {
	apiClient, err := NewClient(serverAddr, authToken, insecure)
	if err != nil {
		return err
	}

	appServiceCloser, appServiceClient, err := apiClient.NewApplicationClient()
	if err != nil {
		return fmt.Errorf("falha ao obter o cliente de aplicação: %w", err)
	}
	defer appServiceCloser.Close()

	// A função GetApplicationDetails ainda é necessária aqui para encontrar o recurso Rollout.
	app, err := GetApplicationDetails(ctx, serverAddr, authToken, insecure, appName)
	if err != nil {
		return fmt.Errorf("não foi possível encontrar a aplicação '%s' para executar a ação '%s': %w", appName, action, err)
	}

	var rolloutResource *v1alpha1.ResourceStatus
	for _, res := range app.Status.Resources {
		if res.Kind == "Rollout" {
			rolloutResource = &res
			break
		}
	}

	if rolloutResource == nil {
		return fmt.Errorf("nenhum recurso 'Rollout' encontrado na aplicação '%s'", appName)
	}

	actionRequest := &application.ResourceActionRunRequest{
		Name:         &appName,
		ResourceName: &rolloutResource.Name,
		Namespace:    &rolloutResource.Namespace,
		Group:        &rolloutResource.Group,
		Version:      &rolloutResource.Version,
		Kind:         &rolloutResource.Kind,
		Action:       &action,
	}

	_, err = appServiceClient.RunResourceAction(ctx, actionRequest)
	if err != nil {
		return fmt.Errorf("falha ao executar a ação '%s' no rollout da aplicação '%s': %w", action, appName, err)
	}

	return nil
}

// PromoteApplicationRollout promove o rollout de uma aplicação para a próxima etapa.
func PromoteApplicationRollout(ctx context.Context, serverAddr, authToken string, insecure bool, appName string) error {
	return runRolloutAction(ctx, serverAddr, authToken, insecure, appName, "promote-full")
}

// AbortApplicationRollout aborta o rollout de uma aplicação.
func AbortApplicationRollout(ctx context.Context, serverAddr, authToken string, insecure bool, appName string) error {
	return runRolloutAction(ctx, serverAddr, authToken, insecure, appName, "abort")
}

// RetryApplicationRollout tenta novamente a última etapa de um rollout que falhou.
func RetryApplicationRollout(ctx context.Context, serverAddr, authToken string, insecure bool, appName string) error {
	return runRolloutAction(ctx, serverAddr, authToken, insecure, appName, "retry")
}
