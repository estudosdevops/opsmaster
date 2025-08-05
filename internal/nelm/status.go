// internal/nelm/status.go
package nelm

import (
	"fmt"
	"strings"
	"time"

	"github.com/estudosdevops/opsmaster/internal/logger"
)

// runNelmForStatus executa o commando nelm para verificar o status de uma release específica
func runNelmForStatus(releaseName, kubeContext, namespace string, timeout time.Duration) error {
	log := logger.Get()

	// Usar timeout padrão se não fornecido
	timeout = getDefaultTimeout(timeout)

	log.Info("Verificando status da release", "release", releaseName, "namespace", namespace)

	// Construir arguments para nelm release list
	args := []string{"release", "list", "--kube-context", kubeContext}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}

	output, err := runNelmCmd(args, ".", true, timeout)
	if err != nil {
		return err
	}

	// Se não há releases, informar
	if strings.Contains(output, "no releases") || strings.TrimSpace(output) == "" {
		log.Info("Nenhuma release encontrada", "namespace", namespace)
		return nil
	}

	// Imprimir a saída capturada
	log.Info("Releases encontradas:", "namespace", namespace)
	fmt.Print(output)

	return nil
}

// ExecuteSmartStatus orquestra a listagem de releases no namespace
func ExecuteSmartStatus(opts StatusOptions) error {
	// Validação das opções obrigatórias
	if opts.ReleaseName == "" {
		return fmt.Errorf("nome da release (ReleaseName) é obrigatório")
	}
	if opts.KubeContext == "" {
		return fmt.Errorf("contexto do Kubernetes (KubeContext) é obrigatório")
	}

	// Se namespace não foi fornecido, usar o nome da release como padrão
	if opts.Namespace == "" {
		opts.Namespace = opts.ReleaseName
	}

	// Listar releases no namespace
	return runNelmForStatus(opts.ReleaseName, opts.KubeContext, opts.Namespace, opts.Timeout)
}
