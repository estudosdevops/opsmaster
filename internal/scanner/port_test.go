package scanner

import (
	"fmt"
	"net"
	"reflect"
	"strconv"
	"testing"
	"time"
)

// TestParsePorts testa a nossa função de análise de portas.
func TestParsePorts(t *testing.T) {
	testCases := []struct {
		name          string
		input         string
		expectedPorts []int
		expectError   bool
	}{
		{
			name:          "Portas simples separadas por vírgula",
			input:         "80,443,8080",
			expectedPorts: []int{80, 443, 8080},
			expectError:   false,
		},
		{
			name:          "Intervalo de portas simples",
			input:         "20-22",
			expectedPorts: []int{20, 21, 22},
			expectError:   false,
		},
		{
			name:          "Combinação de portas e intervals",
			input:         "80, 443, 1020-1022",
			expectedPorts: []int{80, 443, 1020, 1021, 1022},
			expectError:   false,
		},
		{
			name:          "Entrada inválida com texto",
			input:         "80,abc,443",
			expectedPorts: nil,
			expectError:   true,
		},
		{
			name:          "Intervalo inválido",
			input:         "100-90",
			expectedPorts: nil,
			expectError:   true,
		},
		{
			name:          "Porta fora do intervalo válido",
			input:         "80, 99999",
			expectedPorts: nil,
			expectError:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ports, err := parsePorts(tc.input)
			if tc.expectError && err == nil {
				t.Errorf("Esperava um erro, mas recebi nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Não esperava um erro, mas recebi: %v", err)
			}
			if !reflect.DeepEqual(ports, tc.expectedPorts) {
				t.Errorf("Lista de portas inesperada. Esperado: %v, Recebido: %v", tc.expectedPorts, ports)
			}
		})
	}
}

// TestIsValidPort testa a nossa função de validação de portas.
func TestIsValidPort(t *testing.T) {
	testCases := []struct {
		name     string
		port     int
		expected bool
	}{
		{name: "Porta válida no limite inferior", port: 1, expected: true},
		{name: "Porta válida no meio", port: 8080, expected: true},
		{name: "Porta válida no limite superior", port: 65535, expected: true},
		{name: "Porta inválida (zero)", port: 0, expected: false},
		{name: "Porta inválida (acima do limite)", port: 65536, expected: false},
		{name: "Porta inválida (negativa)", port: -1, expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isValidPort(tc.port)
			if result != tc.expected {
				t.Errorf("Resultado inesperado para a porta %d. Esperado: %v, Recebido: %v", tc.port, tc.expected, result)
			}
		})
	}
}

// TestScanPort testa a nossa função de escaneamento de portas.
func TestScanPort(t *testing.T) {
	// Criamos um servidor TCP "fake" que escuta em uma porta aleatória.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Não foi possível criar o servidor de teste: %v", err)
	}
	defer listener.Close()

	// Iniciamos uma goroutine para aceitar a conexão.
	go func() {
		conn, _ := listener.Accept()
		if conn != nil {
			conn.Close()
		}
	}()

	// Aumenta a pausa para dar mais tempo ao sistema operational
	// de preparar o listener, tornando o teste menos propenso a falhas.
	time.Sleep(100 * time.Millisecond)

	openPort := listener.Addr().(*net.TCPAddr).Port

	// --- Tabela de Testes ---
	testCases := []struct {
		name           string
		port           int
		expectedStatus string
	}{
		{
			name:           "Porta aberta",
			port:           openPort,
			expectedStatus: "Aberta",
		},
		{
			name:           "Porta fechada",
			port:           getClosedPort(t),
			expectedStatus: "Fechada",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := scanPort("127.0.0.1", tc.port, 1*time.Second)
			if result.Status != tc.expectedStatus {
				t.Errorf("Status inesperado para a porta %d. Esperado: %s, Recebido: %s", tc.port, tc.expectedStatus, result.Status)
			}
		})
	}
}

// getClosedPort é uma função helper para encontrar uma porta garantidamente fechada.
func getClosedPort(t *testing.T) int {
	// Pede ao sistema operational uma porta livre.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Não foi possível encontrar uma porta livre para o teste: %v", err)
	}
	// Fecha a porta imediatamente, garantindo que ela está livre, mas fechada.
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// TestScanPorts testa a nossa função principal de escaneamento concorrente.
func TestScanPorts(t *testing.T) {
	// Criamos um servidor "fake" para ter uma porta aberta conhecida.
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	defer listener.Close()
	go func() {
		conn, _ := listener.Accept()
		if conn != nil {
			conn.Close()
		}
	}()
	time.Sleep(100 * time.Millisecond)
	openPort := listener.Addr().(*net.TCPAddr).Port
	closedPort := getClosedPort(t)

	testCases := []struct {
		name              string
		portRangeInput    string
		expectedOpenCount int
		expectedTotal     int
	}{
		{
			name:              "Escaneia uma porta aberta e uma fechada",
			portRangeInput:    fmt.Sprintf("%d,%d", openPort, closedPort),
			expectedOpenCount: 1,
			expectedTotal:     2,
		},
		{
			name:              "Escaneia apenas uma porta aberta",
			portRangeInput:    strconv.Itoa(openPort),
			expectedOpenCount: 1,
			expectedTotal:     1,
		},
		{
			name:              "Escaneia apenas uma porta fechada",
			portRangeInput:    strconv.Itoa(closedPort),
			expectedOpenCount: 0,
			expectedTotal:     1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// --- Execução do Teste ---
			results, err := ScanPorts("127.0.0.1", tc.portRangeInput, 1*time.Second)

			// --- Verificações (Assertions) ---
			if err != nil {
				t.Fatalf("A função ScanPorts retornou um erro inesperado: %v", err)
			}

			if len(results) != tc.expectedTotal {
				t.Fatalf("Esperava %d resultados, mas recebi %d", tc.expectedTotal, len(results))
			}

			openCount := 0
			for _, res := range results {
				if res.Status == "Aberta" {
					openCount++
				}
			}

			if openCount != tc.expectedOpenCount {
				t.Errorf("Esperava encontrar %d portas abertas, mas encontrei %d", tc.expectedOpenCount, openCount)
			}
		})
	}
}
