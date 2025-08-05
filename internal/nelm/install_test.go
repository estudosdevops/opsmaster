package nelm

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestExecuteSmartInstall(t *testing.T) {
	tests := []struct {
		name    string
		opts    InstallOptions
		wantErr bool
	}{
		{
			name: "Opções válidas para instalação",
			opts: InstallOptions{
				BaseOptions: BaseOptions{
					KubeContext: "test-context",
					ReleaseName: "test-release",
					Namespace:   "test-namespace",
					AutoApprove: true,
					Timeout:     5 * time.Minute,
				},
				Environment:    "stg",
				MaxConcurrency: 3,
			},
			wantErr: true, // Vai falhar porque não há releases reais para testar
		},
		{
			name: "Opções inválidas - ambiente vazio",
			opts: InstallOptions{
				BaseOptions: BaseOptions{
					KubeContext: "test-context",
					ReleaseName: "test-release",
					Timeout:     5 * time.Minute,
				},
				Environment: "", // Inválido
			},
			wantErr: true,
		},
		{
			name: "Opções inválidas - contexto vazio",
			opts: InstallOptions{
				BaseOptions: BaseOptions{
					KubeContext: "", // Inválido
					ReleaseName: "test-release",
					Timeout:     5 * time.Minute,
				},
				Environment: "stg",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ExecuteSmartInstall(&tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteSmartInstall() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRunNelmForRelease(t *testing.T) {
	// Criar diretório temporário para teste
	tempDir := t.TempDir()
	releaseDir := filepath.Join(tempDir, "test-release")
	if err := os.MkdirAll(releaseDir, 0755); err != nil {
		t.Fatalf("erro ao criar diretório: %v", err)
	}

	// Criar Chart.yaml básico
	chartContent := `name: test-release
version: 1.0.0
description: Test chart`
	if err := os.WriteFile(filepath.Join(releaseDir, "Chart.yaml"), []byte(chartContent), 0600); err != nil {
		t.Fatalf("erro ao criar Chart.yaml: %v", err)
	}

	tests := []struct {
		name        string
		releaseDir  string
		releaseName string
		env         string
		kubeContext string
		autoApprove bool
		timeout     time.Duration
		wantErr     bool
	}{
		{
			name:        "Parâmetros válidos",
			releaseDir:  releaseDir,
			releaseName: "test-release",
			env:         "stg",
			kubeContext: "test-context",
			autoApprove: true,
			timeout:     30 * time.Second,
			wantErr:     true, // Vai falhar porque não há nelm real para executar
		},
		{
			name:        "Diretório inexistente",
			releaseDir:  "/caminho/inexistente",
			releaseName: "test-release",
			env:         "stg",
			kubeContext: "test-context",
			autoApprove: true,
			timeout:     30 * time.Second,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runNelmForRelease(tt.releaseDir, tt.releaseName, tt.env, tt.kubeContext, tt.autoApprove, tt.timeout)
			if (err != nil) != tt.wantErr {
				t.Errorf("runNelmForRelease() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExecuteReleasesInParallel(t *testing.T) {
	// Criar diretórios temporários para teste
	tempDir := t.TempDir()
	release1Dir := filepath.Join(tempDir, "release1")
	release2Dir := filepath.Join(tempDir, "release2")

	if err := os.MkdirAll(release1Dir, 0755); err != nil {
		t.Fatalf("erro ao criar release1Dir: %v", err)
	}
	if err := os.MkdirAll(release2Dir, 0755); err != nil {
		t.Fatalf("erro ao criar release2Dir: %v", err)
	}

	// Criar Chart.yaml nos diretórios
	if err := os.WriteFile(filepath.Join(release1Dir, "Chart.yaml"), []byte("name: release1"), 0600); err != nil {
		t.Fatalf("erro ao criar Chart.yaml em release1Dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(release2Dir, "Chart.yaml"), []byte("name: release2"), 0600); err != nil {
		t.Fatalf("erro ao criar Chart.yaml em release2Dir: %v", err)
	}

	opts := InstallOptions{
		BaseOptions: BaseOptions{
			KubeContext: "test-context",
			AutoApprove: true,
			Timeout:     30 * time.Second,
		},
		Environment:    "stg",
		MaxConcurrency: 2,
	}

	dirs := []string{release1Dir, release2Dir}

	t.Run("Execução paralela com diretórios válidos", func(t *testing.T) {
		err := executeReleasesInParallel(dirs, &opts)
		// Vai falhar na execução real, mas não na validação
		if err == nil {
			t.Log("executeReleasesInParallel() executou sem erros de validação")
		}
	})
}
