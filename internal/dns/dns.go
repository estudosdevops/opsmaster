// opsmaster/internal/dns/dns.go
package dns

import (
	"fmt"
	"strings"

	"github.com/miekg/dns"
)

// Query realiza uma consulta DNS específica para um domínio e tipo de registro.
// É uma versão mais poderosa, similar ao que uma ferramenta como 'dig' faz.
// Retorna uma fatia de strings com os resultados ou um erro.
func Query(domain string, recordType string) ([]string, error) {
	// Configura o cliente DNS que fará a requisição.
	client := new(dns.Client)

	// Configura a mensagem de consulta que será enviada.
	msg := new(dns.Msg)

	// Garante que o domínio termine com um ponto, que é o padrão para um FQDN (Nome de Domínio Totalmente Qualificado).
	fqdn := dns.Fqdn(domain)
	// Converte a string do tipo de registro (ex: "MX") para o tipo numérico que a biblioteca DNS usa.
	qType := dns.StringToType[strings.ToUpper(recordType)]
	if qType == 0 {
		return nil, fmt.Errorf("tipo de registro DNS inválido: %s", recordType)
	}

	// Define a pergunta na mensagem: "Quais são os registros do tipo X para o domínio Y?"
	msg.SetQuestion(fqdn, qType)
	msg.RecursionDesired = true // Pede ao servidor DNS para fazer a busca completa por nós.

	// Envia a consulta para um servidor DNS público conhecido (Google).
	// Adicionamos a porta padrão de DNS (:53).
	response, _, err := client.Exchange(msg, "8.8.8.8:53")
	if err != nil {
		return nil, fmt.Errorf("falha ao se comunicar com o servidor DNS: %w", err)
	}

	// Verifica se a resposta do servidor indica sucesso.
	if response.Rcode != dns.RcodeSuccess {
		return nil, fmt.Errorf("o servidor DNS retornou um erro: %s", dns.RcodeToString[response.Rcode])
	}

	var results []string
	// Itera sobre a seção de respostas da mensagem.
	for _, answer := range response.Answer {
		// O método .String() de cada registro geralmente fornece uma boa representação.
		// Vamos extrair apenas o conteúdo principal do registro.
		// Ex: "google.com. 300 IN A 172.217.29.14" -> extrai "172.217.29.14"
		// Ex: "google.com. 600 IN MX 10 smtp.google.com." -> extrai "10 smtp.google.com."
		parts := strings.Fields(answer.String())
		if len(parts) > 4 {
			// Junta todos os campos após o tipo de registro (ex: MX, que tem prioridade e host)
			resultData := strings.Join(parts[4:], " ")
			results = append(results, resultData)
		}
	}

	return results, nil
}
