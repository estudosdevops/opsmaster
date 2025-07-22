package argocd

import (
	"context"
	"testing"
	"time"
)

// TestProjectLifecycle testa o ciclo de vida completo de um projeto.
func TestProjectLifecycle(t *testing.T) {
	// Pula o teste se as credenciais não foram fornecidas.
	if !runIntegrationTests {
		t.Skip("Pulando teste de integração.")
	}

	projectName := "opsmaster-test-project"
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	t.Cleanup(func() {
		_ = DeleteProject(ctx, testServerAddr, testAuthToken, true, projectName)
	})

	err := CreateProject(ctx, testServerAddr, testAuthToken, true, projectName, "Projeto de teste", []string{"*"})
	if err != nil {
		t.Fatalf("Falha ao criar o projeto de teste: %v", err)
	}

	_, err = GetProject(ctx, testServerAddr, testAuthToken, true, projectName)
	if err != nil {
		t.Fatalf("Não foi possível encontrar o projeto de teste após a criação: %v", err)
	}
}
