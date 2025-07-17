package ip

import (
	"net"
	"testing"
)

// TestFetchPublicIP testa se a função consegue buscar um IP público.
// Este teste requer uma conexão ativa com a internet.
func TestFetchPublicIP(t *testing.T) {
	// Executa a função que queremos testar.
	ipInfo, err := FetchPublicIP()

	// Verifica se houve um erro inesperado.
	if err != nil {
		t.Fatalf("A função FetchPublicIP retornou um erro inesperado: %v", err)
	}

	// Verifica se a struct retornada não é nula.
	if ipInfo == nil {
		t.Fatal("A função FetchPublicIP retornou uma struct nula, mas não um erro.")
	}

	// Verifica se o campo de IP não está vazio e se é um endereço de IP válido.
	if ipInfo.IP == "" {
		t.Error("O IP público retornado está vazio.")
	}
	if net.ParseIP(ipInfo.IP) == nil {
		t.Errorf("O valor retornado '%s' não é um endereço de IP válido.", ipInfo.IP)
	}
}

// TestFetchLocalNetworkInfo testa se a função consegue buscar informações da rede local.
// Este teste assume que a máquina que o executa tem pelo menos uma interface de rede ativa.
func TestFetchLocalNetworkInfo(t *testing.T) {
	// Executa a função que queremos testar.
	localInfo, err := FetchLocalNetworkInfo()

	// Verifica se houve um erro inesperado.
	// Usamos t.Skip se não houver rede, para não falhar em ambientes de CI.
	if err != nil {
		t.Skipf("Não foi possível executar o teste de rede local, talvez não haja uma interface de rede ativa: %v", err)
	}

	// Verifica se a struct retornada não é nula.
	if localInfo == nil {
		t.Fatal("A função FetchLocalNetworkInfo retornou uma struct nula, mas não um erro.")
	}

	// Verifica se os campos principais não estão vazios.
	if localInfo.IPAddress == "" {
		t.Error("O endereço de IP local retornado está vazio.")
	}
	if localInfo.InterfaceName == "" {
		t.Error("O nome da interface de rede retornado está vazio.")
	}
}
