// internal/monitor/http_monitor.go
package monitor

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/estudosdevops/opsmaster/internal/logger"
)

// defaultCheckTimeout define o tempo limite padrão para cada verificação HTTP.
const defaultCheckTimeout = 10 * time.Second

// checkURL realiza uma única verificação HTTP na URL fornecida.
func checkURL(ctx context.Context, url string) string {
	startTime := time.Now()
	client := http.Client{}

	// Cria a requisição com o contexto.
	req, err := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
	if err != nil {
		return fmt.Sprintf("FALHA: %s - Erro ao criar requisição: %v", url, err)
	}

	resp, err := client.Do(req)
	latency := time.Since(startTime)

	// Se houver um erro de conexão, é uma falha.
	if err != nil {
		return fmt.Sprintf("FALHA: %s - Erro de conexão: %v", url, err)
	}
	defer resp.Body.Close()

	// Verifica se o status code é de sucesso (2xx).
	// Qualquer outro status code é considerado uma falha para o monitor.
	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		return fmt.Sprintf("SUCESSO: %s - Status: %s (%.2fms)", url, resp.Status, float64(latency.Milliseconds()))
	}

	return fmt.Sprintf("FALHA: %s - Status: %s (%.2fms)", url, resp.Status, float64(latency.Milliseconds()))
}

// StartMonitoring inicia o processo de monitoramento para uma URL.
func StartMonitoring(url string, interval time.Duration, count int) {
	log := logger.Get()
	log.Info("Iniciando monitoramento", "url", url, "intervalo", interval)
	if count > 0 {
		log.Info("O monitoramento será executado", "vezes", count)
	} else {
		log.Info("Pressione CTRL+C para parar.")
	}

	// Função helper para executar um único check.
	runCheck := func() {
		// Cria um contexto com um timeout para cada chamada individual.
		ctx, cancel := context.WithTimeout(context.Background(), defaultCheckTimeout)
		defer cancel()
		result := checkURL(ctx, url)
		log.Info(result)
	}

	// Executa uma vez imediatamente antes de iniciar o loop.
	runCheck()

	if count == 1 {
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	checksDone := 1
	for range ticker.C {
		runCheck()
		if count > 0 {
			checksDone++
			if checksDone >= count {
				break
			}
		}
	}
}
