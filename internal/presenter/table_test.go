package presenter

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// TestPrintTable testa a nossa função de impressão de tabelas.
func TestPrintTable(t *testing.T) {
	// Esta é a parte mais importante: vamos capturar a saída padrão (stdout).
	oldStdout := os.Stdout // Guarda a saída padrão original.
	r, w, _ := os.Pipe()   // Cria um "cano" (pipe) para onde a saída será redirecionada.
	os.Stdout = w          // Redireciona a saída padrão para o nosso cano.

	header := []string{"NOME", "STATUS"}
	rows := [][]string{
		{"servico-a", "Healthy"},
		{"servico-b", "Degraded"},
	}
	PrintTable(header, rows) // Chama a função que queremos testar.

	// --- Verificações (Assertions) ---
	w.Close()             // Fecha o lado de escrita do cano.
	os.Stdout = oldStdout // Restaura a saída padrão original.
	var buf bytes.Buffer
	io.Copy(&buf, r) // Lê tudo o que foi escrito no cano para um buffer.

	output := buf.String()

	// Verifica se a saída contém os elementos que esperamos.
	if !strings.Contains(output, "NOME") || !strings.Contains(output, "STATUS") {
		t.Errorf("O cabeçalho não foi impresso corretamente. Saída:\n%s", output)
	}
	if !strings.Contains(output, "servico-a") || !strings.Contains(output, "Healthy") {
		t.Errorf("A primeira linha não foi impressa corretamente. Saída:\n%s", output)
	}
	if !strings.Contains(output, "servico-b") || !strings.Contains(output, "Degraded") {
		t.Errorf("A segunda linha não foi impressa corretamente. Saída:\n%s", output)
	}
}
