// internal/argocd/cluster.go
package argocd

import (
	"context"
	"fmt"

	"github.com/argoproj/argo-cd/v2/pkg/apiclient/cluster"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
)

// ListClusters busca todos os clusters Kubernetes registados no Argo CD.
func ListClusters(ctx context.Context, serverAddr, authToken string, insecure bool) (*v1alpha1.ClusterList, error) {
	apiClient, err := NewClient(serverAddr, authToken, insecure)
	if err != nil {
		return nil, err
	}

	clusterServiceCloser, clusterServiceClient, err := apiClient.NewClusterClient()
	if err != nil {
		return nil, fmt.Errorf("falha ao obter o cliente de cluster: %w", err)
	}
	defer clusterServiceCloser.Close()

	return clusterServiceClient.List(ctx, &cluster.ClusterQuery{})
}
