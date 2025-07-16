package dns

import (
	"testing"
)

// TestQuery testa a nossa função de consulta DNS.
func TestQuery(t *testing.T) {
	testCases := []struct {
		name         string
		domain       string
		recordType   string
		expectError  bool
		expectResult bool
	}{
		{
			name:         "Registro A para domínio válido",
			domain:       "google.com",
			recordType:   "A",
			expectError:  false,
			expectResult: true,
		},
		{
			name:         "Registro MX para domínio válido",
			domain:       "google.com",
			recordType:   "MX",
			expectError:  false,
			expectResult: true,
		},
		{
			name:         "Registro CNAME para um subdomínio conhecido",
			domain:       "www.github.com",
			recordType:   "CNAME",
			expectError:  false,
			expectResult: true,
		},
		{
			name:         "Domínio que não existe",
			domain:       "um-dominio-que-provavelmente-nao-existe.com",
			recordType:   "A",
			expectError:  true,
			expectResult: false,
		},
		{
			name:         "Tipo de registro inválido",
			domain:       "google.com",
			recordType:   "INVALIDO",
			expectError:  true,
			expectResult: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			results, err := Query(tc.domain, tc.recordType)

			if tc.expectError && err == nil {
				t.Errorf("Esperava um erro para o domínio '%s' com tipo '%s', mas não recebi", tc.domain, tc.recordType)
			}
			if !tc.expectError && err != nil {
				t.Errorf("Não esperava um erro para o domínio '%s' com tipo '%s', mas recebi: %v", tc.domain, tc.recordType, err)
			}

			// Verifica se a expectativa de resultado corresponde à realidade.
			if tc.expectResult && len(results) == 0 {
				t.Errorf("Esperava pelo menos um resultado para o domínio '%s' com tipo '%s', mas não recebi nenhum", tc.domain, tc.recordType)
			}
			if !tc.expectResult && len(results) > 0 {
				t.Errorf("Não esperava nenhum resultado, mas recebi: %v", results)
			}
		})
	}
}
