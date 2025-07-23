package argocd

import (
	"context"
	"testing"
	"time"
)

// TestListClusters testa a função ListClusters.
func TestListClusters(t *testing.T) {
	if !runIntegrationTests {
		t.Skip("Pulando teste de integração de listagem de clusters.")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	clusterList, err := ListClusters(ctx, testServerAddr, testAuthToken, true)
	if err != nil {
		t.Fatalf("Falha ao listar clusters: %v", err)
	}

	if len(clusterList.Items) == 0 {
		t.Error("Esperava encontrar pelo menos um cluster (o in-cluster), mas a lista está vazia.")
	}
	t.Logf("Encontrados %d clusters.", len(clusterList.Items))
}
