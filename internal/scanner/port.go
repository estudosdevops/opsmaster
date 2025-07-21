package scanner

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ScanResult armazena o resultado da verificação de uma única porta.
type ScanResult struct {
	Port   int    // Número da porta verificada.
	Status string // Status da porta (aberta, fechada, filtrada).
}

// isValidPort verifica se um número de porta está no intervalo válido (1-65535).
func isValidPort(port int) bool {
	return port >= 1 && port <= 65535
}

// scanPort tenta se conectar a uma única porta e retorna um resultado detalhado.
func scanPort(host string, port int, timeout time.Duration) ScanResult {
	address := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", address, timeout)

	if err != nil {
		// Analisa o tipo de erro para dar um resultado mais preciso.
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return ScanResult{Port: port, Status: "Timeout/Filtrado"}
		}
		// "connection refused" significa que a porta está ativa, mas fechada.
		if strings.Contains(err.Error(), "connection refused") {
			return ScanResult{Port: port, Status: "Fechada"}
		}
		// Outros erros podem indicar problems de resolução de nome ou de rota.
		// Para simplificar, vamos considerá-los como filtrados também.
		return ScanResult{Port: port, Status: "Fechado/Filtrado"}
	}

	// Se a conexão foi bem-sucedida, a porta está aberta.
	conn.Close()
	return ScanResult{Port: port, Status: "Aberta"}
}

// parsePorts interpreta a string de portas e agora valida o intervalo numérico.
func parsePorts(portRange string) ([]int, error) {
	var ports []int
	parts := strings.Split(portRange, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("intervalo de portas inválido: %s", part)
			}
			start, err1 := strconv.Atoi(rangeParts[0])
			end, err2 := strconv.Atoi(rangeParts[1])
			if err1 != nil || err2 != nil || start > end {
				return nil, fmt.Errorf("intervalo de portas numérico inválido: %s", part)
			}

			// Verifica se os números do intervalo são válidos.
			if !isValidPort(start) || !isValidPort(end) {
				return nil, fmt.Errorf("número de porta fora do intervalo válido (1-65535): %s", part)
			}

			for i := start; i <= end; i++ {
				ports = append(ports, i)
			}
		} else {
			port, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("porta inválida: %s", part)
			}

			// Verifica se a porta única é válida.
			if !isValidPort(port) {
				return nil, fmt.Errorf("número de porta fora do intervalo válido (1-65535): %s", part)
			}
			ports = append(ports, port)
		}
	}
	return ports, nil
}

// ScanPorts executa o escaneamento de portas de forma concorrente.
func ScanPorts(host, portRange string, timeout time.Duration) ([]ScanResult, error) {
	portsToScan, err := parsePorts(portRange)
	if err != nil {
		return nil, err
	}

	// Canal para coletar resultados de forma concorrente.
	resultsChan := make(chan ScanResult, len(portsToScan))
	var wg sync.WaitGroup

	for _, port := range portsToScan {
		wg.Add(1)
		go func(p int) {
			defer wg.Done()
			result := scanPort(host, p, timeout)
			resultsChan <- result // Envia o resultado para o canal.
		}(port)
	}

	// Espera todas as goroutines terminarem.
	wg.Wait()
	// Fecha o canal para sinalizar que não haverá mais resultados.
	close(resultsChan)

	var finalResults []ScanResult
	// Coleta todos os resultados do canal.
	for result := range resultsChan {
		finalResults = append(finalResults, result)
	}

	return finalResults, nil
}
