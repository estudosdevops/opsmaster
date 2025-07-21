// opsmaster/internal/ip/ip.go
package ip

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/jackpal/gateway"
)

// LocalNetworkInfo agrupa as informações de uma interface de rede local.
type LocalNetworkInfo struct {
	InterfaceName string
	IPAddress     string
	SubnetMask    string
	MACAddress    string
	Gateway       string
}

// PublicIPInfo agrupa as informações de IP público retornadas pela API ipinfo.io.
// As tags `json:"..."` dizem ao Go como mapear os campos da resposta JSON para esta struct.
type PublicIPInfo struct {
	IP       string `json:"ip"`
	City     string `json:"city"`
	Region   string `json:"region"`
	Country  string `json:"country"`
	Org      string `json:"org"`
	Timezone string `json:"timezone"`
}

// FetchPublicIP busca o endereço de IP público fazendo uma requisição a um serviço externo.
// Agora aceita um contexto para controle de timeout e cancelamento.
func FetchPublicIP(ctx context.Context) (*PublicIPInfo, error) {
	const serviceURL = "https://ipinfo.io/json"
	client := http.Client{
		// O timeout do cliente ainda é uma boa prática como um fallback geral.
		Timeout: 10 * time.Second,
	}

	// Cria a requisição com o contexto.
	req, err := http.NewRequestWithContext(ctx, "GET", serviceURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("falha ao criar a requisição HTTP: %w", err)
	}

	// Executa a requisição usando o método Do.
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("falha ao conectar ao serviço de IP: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("serviço de IP retornou um status inesperado: %s", resp.Status)
	}

	var ipInfo PublicIPInfo
	if err := json.NewDecoder(resp.Body).Decode(&ipInfo); err != nil {
		return nil, fmt.Errorf("falha ao decodificar a resposta JSON: %w", err)
	}

	return &ipInfo, nil
}

// FetchLocalNetworkInfo busca o endereço de IP local, o gateway padrão e a interface de saída.
func FetchLocalNetworkInfo() (*LocalNetworkInfo, error) {
	// Descobre o endereço do gateway padrão.
	gatewayIP, err := gateway.DiscoverGateway()
	if err != nil {
		return nil, fmt.Errorf("falha ao descobrir o gateway: %w", err)
	}

	localIP, err := gateway.DiscoverInterface()
	if err != nil {
		return nil, fmt.Errorf("falha ao descobrir a interface de saída: %w", err)
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				if ipnet.IP.Equal(localIP) {
					// Converte a mascara de rede para o formato de string (ex: 255.255.255.0)
					mask := fmt.Sprintf("%d.%d.%d.%d", ipnet.Mask[0], ipnet.Mask[1], ipnet.Mask[2], ipnet.Mask[3])

					// Encontramos a interface correta, agora podemos retornar todas as informações.
					return &LocalNetworkInfo{
						InterfaceName: i.Name,
						IPAddress:     localIP.String(),
						SubnetMask:    mask,
						MACAddress:    i.HardwareAddr.String(),
						Gateway:       gatewayIP.String(),
					}, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("não foi possível encontrar o nome da interface para o IP %s", localIP)
}
