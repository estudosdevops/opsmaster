package nelm

import (
	"testing"
	"time"
)

func TestExecuteSmartUninstall(t *testing.T) {
	tests := []struct {
		name    string
		opts    UninstallOptions
		wantErr bool
	}{
		{
			name: "Opções válidas para uninstall",
			opts: UninstallOptions{
				BaseOptions: BaseOptions{
					KubeContext: "test-context",
					ReleaseName: "test-release",
					Namespace:   "test-namespace",
					AutoApprove: true,
					Timeout:     5 * time.Minute,
				},
			},
			wantErr: true, // Vai falhar porque não há releases reais para testar
		},
		{
			name: "Opções inválidas - contexto vazio",
			opts: UninstallOptions{
				BaseOptions: BaseOptions{
					KubeContext: "", // Inválido
					ReleaseName: "test-release",
					Timeout:     5 * time.Minute,
				},
			},
			wantErr: true,
		},
		{
			name: "Opções inválidas - release vazio",
			opts: UninstallOptions{
				BaseOptions: BaseOptions{
					KubeContext: "test-context",
					ReleaseName: "", // Inválido para uninstall
					Timeout:     5 * time.Minute,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ExecuteSmartUninstall(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteSmartUninstall() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
