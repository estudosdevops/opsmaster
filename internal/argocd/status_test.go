package argocd

import (
	"context"
	"testing"
	"time"
)

// setupTestApp é uma função helper que cria uma aplicação de teste e garante a sua limpeza.
func setupTestApp(t *testing.T) (*AppOptions, context.Context) {
	if !runIntegrationTests {
		t.Skip("Pulando teste de integração de status.")
	}

	opts := &AppOptions{
		AppName:        "opsmaster-test-status",
		Project:        "default",
		DestinationNS:  "default",
		RepoURL:        "https://github.com/argoproj/argo-cd.git",
		RepoPath:       "applications/guestbook",
		TargetRevision: "HEAD",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)

	// Cria a aplicação antes dos testes de leitura.
	err := CreateApplication(ctx, testServerAddr, testAuthToken, true, opts)
	if err != nil {
		cancel()
		t.Fatalf("Falha ao criar a aplicação de setup para os testes de status: %v", err)
	}

	// Agenda a limpeza, incluindo a chamada a 'cancel'.
	t.Cleanup(func() {
		_ = DeleteApplication(ctx, testServerAddr, testAuthToken, true, opts.AppName)
		cancel()
	})

	return opts, ctx
}

func TestGetApplication(t *testing.T) {
	opts, ctx := setupTestApp(t)

	appStatus, err := GetApplication(ctx, testServerAddr, testAuthToken, true, opts.AppName)
	if err != nil {
		t.Fatalf("Não foi possível encontrar a aplicação de teste: %v", err)
	}
	if appStatus.Name != opts.AppName {
		t.Errorf("O nome da aplicação encontrada ('%s') não corresponde ao esperado ('%s').", appStatus.Name, opts.AppName)
	}
}

func TestListApplications(t *testing.T) {
	opts, ctx := setupTestApp(t)

	apps, err := ListApplications(ctx, testServerAddr, testAuthToken, true)
	if err != nil {
		t.Fatalf("Falha ao listar as aplicações: %v", err)
	}

	found := false
	for _, app := range apps {
		if app.Name == opts.AppName {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("A aplicação de teste '%s' não foi encontrada na lista de aplicações.", opts.AppName)
	}
}
