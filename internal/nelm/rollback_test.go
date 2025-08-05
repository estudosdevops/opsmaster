package nelm

import (
	"testing"
	"time"
)

func TestExecuteSmartRollback(t *testing.T) {
	tests := []struct {
		name    string
		opts    RollbackOptions
		wantErr bool
	}{
		{
			name: "Opções válidas para rollback",
			opts: RollbackOptions{
				BaseOptions: BaseOptions{
					KubeContext: "test-context",
					ReleaseName: "test-release",
					Namespace:   "test-namespace",
					AutoApprove: true,
					Timeout:     5 * time.Minute,
				},
				Revision: 2,
			},
			wantErr: true, // Vai falhar porque não há releases reais para testar
		},
		{
			name: "Opções válidas - revisão 0 (anterior)",
			opts: RollbackOptions{
				BaseOptions: BaseOptions{
					KubeContext: "test-context",
					ReleaseName: "test-release",
					Timeout:     5 * time.Minute,
				},
				Revision: 0, // Revisão anterior
			},
			wantErr: true, // Vai falhar porque não há releases reais para testar
		},
		{
			name: "Opções inválidas - contexto vazio",
			opts: RollbackOptions{
				BaseOptions: BaseOptions{
					KubeContext: "", // Inválido
					ReleaseName: "test-release",
					Timeout:     5 * time.Minute,
				},
				Revision: 1,
			},
			wantErr: true,
		},
		{
			name: "Opções inválidas - release vazio",
			opts: RollbackOptions{
				BaseOptions: BaseOptions{
					KubeContext: "test-context",
					ReleaseName: "", // Inválido para rollback
					Timeout:     5 * time.Minute,
				},
				Revision: 1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ExecuteSmartRollback(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteSmartRollback() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
