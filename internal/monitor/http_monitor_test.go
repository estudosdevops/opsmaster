// internal/monitor/http_monitor_test.go
package monitor

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestCheckURL testa a nossa função de verificação de URL.
func TestCheckURL(t *testing.T) {
	// Cria um servidor de teste que responderá às nossas requisições.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/success" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, "Tudo certo!")
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, "Deu erro!")
		}
	}))
	defer server.Close()

	// Tabela de Testes
	testCases := []struct {
		name           string
		urlToTest      string
		expectedResult string
	}{
		{
			name:           "URL com sucesso",
			urlToTest:      server.URL + "/success",
			expectedResult: "SUCESSO",
		},
		{
			name:           "URL com falha (erro de servidor)",
			urlToTest:      server.URL + "/failure",
			expectedResult: "FALHA",
		},
		{
			name:           "URL que não existe (erro de conexão)",
			urlToTest:      "http://localhost:12345",
			expectedResult: "FALHA",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := checkURL(tc.urlToTest)
			if !strings.Contains(result, tc.expectedResult) {
				t.Errorf("Resultado inesperado. Esperava conter '%s', mas recebi: '%s'", tc.expectedResult, result)
			}
		})
	}
}
