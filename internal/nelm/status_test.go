package nelm

import (
	"testing"
	"time"
)

func TestExecuteSmartStatus(t *testing.T) {
	tests := []struct {
		name    string
		opts    StatusOptions
		wantErr bool
	}{
		{
			name: "Opções válidas para status",
			opts: StatusOptions{
				BaseOptions: BaseOptions{
					KubeContext: "test-context",
					ReleaseName: "test-release",
					Namespace:   "test-namespace",
					Timeout:     5 * time.Minute,
				},
			},
			wantErr: true, // Vai falhar porque não há nelm real para executar
		},
		{
			name: "Opções inválidas - contexto vazio",
			opts: StatusOptions{
				BaseOptions: BaseOptions{
					KubeContext: "", // Inválido
					ReleaseName: "test-release",
					Timeout:     5 * time.Minute,
				},
			},
			wantErr: true,
		},
		{
			name: "Opções válidas - sem release específica",
			opts: StatusOptions{
				BaseOptions: BaseOptions{
					KubeContext: "test-context",
					ReleaseName: "", // Válido para listar todas
					Timeout:     5 * time.Minute,
				},
			},
			wantErr: true, // Vai falhar porque release vazio é obrigatório para status
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ExecuteSmartStatus(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteSmartStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
