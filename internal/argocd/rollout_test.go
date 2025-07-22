package argocd

import (
	"context"
	"testing"
	"time"
)

// TestRolloutActions testa as ações de rollout (promote, abort, retry).
// Este é um teste mais complexo que requer uma aplicação com um Rollout configurado.
func TestRolloutActions(t *testing.T) {
	// Pula o teste se as credenciais não foram fornecidas.
	if !runIntegrationTests {
		t.Skip("Pulando teste de integração.")
	}

	// Para este teste, assumimos que já existe uma aplicação chamada 'sample-api-stg'
	// que está usando um Rollout e está no estado 'Paused'.
	// Em um cenário de CI real, a primeira parte do teste criaria esta aplicação.
	appName := "sample-api-stg"

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Testa a ação de promover
	t.Run("PromoteRollout", func(t *testing.T) {
		err := PromoteApplicationRollout(ctx, testServerAddr, testAuthToken, true, appName)
		if err != nil {
			t.Errorf("A função PromoteApplicationRollout retornou um erro inesperado: %v", err)
		}
	})

	// NOTA: Testes para 'abort' e 'retry' seguiriam um padrão semelhante, mas exigiriam
	// que a aplicação estivesse em um estado 'Degraded' ou em andamento para serem válidos.
}
