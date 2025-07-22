package argocd

import (
	"context"
	"testing"
	"time"
)

// TestRepositoryLifecycle testa o ciclo de vida completo de um repositório (adicionar e apagar).
func TestRepositoryLifecycle(t *testing.T) {
	// Pula o teste se as credenciais não foram fornecidas.
	if !runIntegrationTests {
		t.Skip("Pulando teste de integração.")
	}

	// Usamos um repositório público bem conhecido para o teste.
	repoURL := "https://github.com/argoproj/argo-helm"
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	// --- Etapa de Limpeza (Teardown) ---
	t.Cleanup(func() {
		t.Logf("Limpando: apagando o repositório de teste '%s'", repoURL)
		_ = DeleteRepository(ctx, testServerAddr, testAuthToken, true, repoURL)
	})

	// --- Execução do Teste ---
	// 1. Tenta adicionar o repositório.
	err := AddRepository(ctx, testServerAddr, testAuthToken, true, repoURL, "", "") // Repositório público, sem credenciais.
	if err != nil {
		t.Fatalf("Falha ao adicionar o repositório de teste: %v", err)
	}
	t.Logf("Repositório de teste '%s' adicionado com sucesso.", repoURL)

	// 2. Tenta buscar o repositório para confirmar que ele existe.
	_, err = GetRepository(ctx, testServerAddr, testAuthToken, true, repoURL)
	if err != nil {
		t.Fatalf("Não foi possível encontrar o repositório de teste após a criação: %v", err)
	}
	t.Log("Repositório de teste encontrado com sucesso após a criação.")
}
