// opsmaster/internal/dns/dns.go
package dns

import (
	"fmt"
	"strings"

	"github.com/miekg/dns"
)

// minDNSRecordParts define o número mínimo de partes que esperamos em uma resposta de registro DNS.
const minDNSRecordParts = 4

// Query realiza uma consulta DNS específica para um domínio e tipo de registro.
func Query(domain, recordType string) ([]string, error) {
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
	for _, answer := range response.Answer {
		parts := strings.Fields(answer.String())
		if len(parts) > minDNSRecordParts {
			resultData := strings.Join(parts[4:], " ")
			results = append(results, resultData)
		}
	}

	return results, nil
}
