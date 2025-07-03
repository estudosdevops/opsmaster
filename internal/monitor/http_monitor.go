// opsmaster/internal/monitor/http_monitor.go
package monitor

import (
	"fmt"
	"net/http"
	"time"

	"github.com/estudosdevops/opsmaster/internal/logger" // Importa nosso novo pacote de logger!
)

// checkURL realiza uma única verificação HTTP.
// Agora, ela retorna os dados de forma estruturada para o logger formatar.
// Retorna: o status HTTP, a latência e um erro (se ocorrer).
func checkURL(url string) (string, time.Duration, error) {
	startTime := time.Now()

	client := http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(url)
	latency := time.Since(startTime)

	if err != nil {
		// Se houver um erro de conexão, retorne o erro.
		return "", 0, err
	}
	defer resp.Body.Close()

	// Se a requisição for bem-sucedida, retorne o status, a latência e nenhum erro (nil).
	return resp.Status, latency, nil
}

// StartMonitoring inicia o processo de monitoramento para uma URL.
func StartMonitoring(url string, interval time.Duration, count int) {
	// Pega a instância do logger do nosso pacote centralizado.
	// Note como este pacote não precisa mais saber sobre 'tint' ou 'os'.
	logger := logger.Get()

	// Usando o logger para as mensagens iniciais.
	logger.Info("Iniciando monitoramento", "url", url, "intervalo", interval)
	if count > 0 {
		logger.Info("O monitoramento tem um número definido de execuções", "execuções", count)
	} else {
		logger.Info("Pressione CTRL+C para parar.")
	}
	fmt.Println("----------------------------------------------------------------------")

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	checksDone := 0
	for {
		<-ticker.C

		// Chama a função de verificação para obter os dados.
		status, latency, err := checkURL(url)

		// Verifica se houve um erro e usa o nível de log apropriado.
		if err != nil {
			logger.Error("Falha na verificação", "url", url, "erro", err)
		} else {
			logger.Info(
				"Verificação bem-sucedida",
				"url", url,
				"status", status,
				"latência", latency.Round(time.Millisecond), // Arredonda a latência para melhor leitura
			)
		}

		if count > 0 {
			checksDone++
			if checksDone >= count {
				fmt.Println("----------------------------------------------------------------------")
				logger.Info("Número de verificações concluído. Encerrando.")
				break
			}
		}
	}
}
