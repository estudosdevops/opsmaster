package argocd

import (
	"context"
	"testing"
	"time"
)

// TestApplicationCreateDelete testa as operações de escrita (Create/Delete) para uma aplicação.
func TestApplicationCreateDelete(t *testing.T) {
	// Pula o teste se as credenciais não foram fornecidas.
	if !runIntegrationTests {
		t.Skip("Pulando teste de integração.")
	}

	opts := &AppOptions{
		AppName:        "opsmaster-test-app-lifecycle",
		Project:        "default",
		DestinationNS:  "default",
		RepoURL:        "https://github.com/argoproj/argo-cd.git",
		RepoPath:       "applications/guestbook",
		TargetRevision: "HEAD",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Etapa de Limpeza
	t.Cleanup(func() {
		_ = DeleteApplication(ctx, testServerAddr, testAuthToken, true, opts.AppName)
	})

	// Testa a Criação
	err := CreateApplication(ctx, testServerAddr, testAuthToken, true, opts)
	if err != nil {
		t.Fatalf("Falha ao criar a aplicação de teste: %v", err)
	}
	t.Logf("Aplicação de teste '%s' criada com sucesso.", opts.AppName)
}
