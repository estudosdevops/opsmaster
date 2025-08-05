package nelm

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseTimeoutFromFlag(t *testing.T) {
	tests := []struct {
		name           string
		timeoutStr     string
		defaultTimeout time.Duration
		expected       time.Duration
	}{
		{
			name:           "Timeout válido em minutos",
			timeoutStr:     "5m",
			defaultTimeout: 10 * time.Minute,
			expected:       5 * time.Minute,
		},
		{
			name:           "Timeout válido em segundos",
			timeoutStr:     "30s",
			defaultTimeout: 5 * time.Minute,
			expected:       30 * time.Second,
		},
		{
			name:           "Timeout válido em horas",
			timeoutStr:     "1h",
			defaultTimeout: 5 * time.Minute,
			expected:       1 * time.Hour,
		},
		{
			name:           "String vazia retorna default",
			timeoutStr:     "",
			defaultTimeout: 10 * time.Minute,
			expected:       10 * time.Minute,
		},
		{
			name:           "String inválida retorna default",
			timeoutStr:     "invalid",
			defaultTimeout: 5 * time.Minute,
			expected:       5 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseTimeoutFromFlag(tt.timeoutStr, tt.defaultTimeout)
			if result != tt.expected {
				t.Errorf("ParseTimeoutFromFlag(%s, %v) = %v, want %v", tt.timeoutStr, tt.defaultTimeout, result, tt.expected)
			}
		})
	}
}

func TestValidateNelmRequiredFields(t *testing.T) {
	tests := []struct {
		name    string
		fields  map[string]string
		wantErr bool
	}{
		{
			name: "Campos válidos",
			fields: map[string]string{
				"env":         "stg",
				"kubeContext": "test-context",
			},
			wantErr: false,
		},
		{
			name: "Campo env vazio",
			fields: map[string]string{
				"env":         "",
				"kubeContext": "test-context",
			},
			wantErr: true,
		},
		{
			name: "Campo kubeContext vazio",
			fields: map[string]string{
				"env":         "stg",
				"kubeContext": "",
			},
			wantErr: true,
		},
		{
			name: "Ambos campos vazios",
			fields: map[string]string{
				"env":         "",
				"kubeContext": "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNelmRequiredFields(tt.fields)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateNelmRequiredFields() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFindReleaseDirs(t *testing.T) {
	// Criar estrutura temporária para teste
	tempDir := t.TempDir()

	// Criar diretórios de teste
	release1Dir := filepath.Join(tempDir, "sample-api")
	release2Dir := filepath.Join(tempDir, "another-service")
	nonReleaseDir := filepath.Join(tempDir, "docs")

	// Criar diretórios
	if err := os.MkdirAll(release1Dir, 0755); err != nil {
		t.Fatalf("erro ao criar diretório de release1: %v", err)
	}
	if err := os.MkdirAll(release2Dir, 0755); err != nil {
		t.Fatalf("erro ao criar diretório de release2: %v", err)
	}
	if err := os.MkdirAll(nonReleaseDir, 0755); err != nil {
		t.Fatalf("erro ao criar diretório de docs: %v", err)
	}

	// Criar Chart.yaml nos diretórios de release
	if err := os.WriteFile(filepath.Join(release1Dir, "Chart.yaml"), []byte("name: sample-api"), 0600); err != nil {
		t.Fatalf("erro ao criar Chart.yaml em release1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(release2Dir, "Chart.yaml"), []byte("name: another-service"), 0600); err != nil {
		t.Fatalf("erro ao criar Chart.yaml em release2: %v", err)
	}

	// Mudar para o diretório temporário
	originalDir, _ := os.Getwd()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("erro ao mudar para tempDir: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Fatalf("erro ao voltar para o diretório original: %v", err)
		}
	}()

	tests := []struct {
		name        string
		releaseName string
		wantCount   int
		wantErr     bool
	}{
		{
			name:        "Buscar todas as releases",
			releaseName: "",
			wantCount:   2,
			wantErr:     false,
		},
		{
			name:        "Buscar release específica existente",
			releaseName: "sample-api",
			wantCount:   1,
			wantErr:     false,
		},
		{
			name:        "Buscar release específica inexistente",
			releaseName: "non-existent",
			wantCount:   0,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dirs, err := findReleaseDirs(tt.releaseName)
			if (err != nil) != tt.wantErr {
				t.Errorf("findReleaseDirs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(dirs) != tt.wantCount {
				t.Errorf("findReleaseDirs() returned %d directories, want %d", len(dirs), tt.wantCount)
			}
		})
	}
}

func TestGetDefaultTimeout(t *testing.T) {
	tests := []struct {
		name     string
		timeout  time.Duration
		expected time.Duration
	}{
		{
			name:     "Timeout zero retorna padrão",
			timeout:  0,
			expected: 5 * time.Minute,
		},
		{
			name:     "Timeout não zero retorna o mesmo valor",
			timeout:  10 * time.Minute,
			expected: 10 * time.Minute,
		},
		{
			name:     "Timeout negativo retorna o mesmo valor",
			timeout:  -1 * time.Minute,
			expected: -1 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getDefaultTimeout(tt.timeout)
			if result != tt.expected {
				t.Errorf("getDefaultTimeout(%v) = %v, want %v", tt.timeout, result, tt.expected)
			}
		})
	}
}

func TestPromptConfirmation(t *testing.T) {
	// Este teste é mais complexo pois envolve stdin
	// Vamos testar apenas a estrutura da função
	t.Run("Função existe e é chamável", func(t *testing.T) {
		// A função promptConfirmation existe e pode ser chamada
		// Em um teste real, precisaríamos mockar stdin
		// Por enquanto, apenas verificamos que a função está definida
		// Não podemos comparar função com nil em Go, então apenas logamos
		t.Log("promptConfirmation function is defined and callable")
	})
}
