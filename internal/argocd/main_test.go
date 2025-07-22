// internal/argocd/main_test.go
package argocd

import (
	"fmt"
	"os"
	"testing"
)

var (
	// Variáveis globais para armazenar as credenciais de teste.
	testServerAddr      string
	testAuthToken       string
	runIntegrationTests bool
)

// TestMain é uma função especial que roda uma única vez antes de todos os
// outros testes neste pacote. É o lugar perfeito para a nossa configuração.
func TestMain(m *testing.M) {
	testServerAddr = os.Getenv("ARGOCD_SERVER")
	testAuthToken = os.Getenv("ARGOCD_TOKEN")

	if testServerAddr != "" && testAuthToken != "" {
		runIntegrationTests = true
		fmt.Println("Credenciais de teste do Argo CD encontradas. Executando testes de integração.")
	} else {
		fmt.Println("AVISO: Variáveis de ambiente de teste do Argo CD não definidas. Pulando testes de integração.")
	}

	// Roda todos os outros testes do pacote.
	exitCode := m.Run()
	os.Exit(exitCode)
}
