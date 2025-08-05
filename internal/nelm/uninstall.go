// internal/nelm/uninstall.go
package nelm

import (
	"fmt"
	"time"

	"github.com/estudosdevops/opsmaster/internal/logger"
)

// runNelmForUninstall executa os commandos nelm para desinstalar uma release específica
func runNelmForUninstall(releaseDir, releaseName, kubeContext string, autoApprove bool, timeout time.Duration) error {
	log := logger.Get()

	// Usar timeout padrão se não fornecido
	timeout = getDefaultTimeout(timeout)

	// 1. Se não for auto-approve, perguntar ao usuário
	if !autoApprove {
		message := fmt.Sprintf("Deseja desinstalar a release '%s'? (s/N): ", releaseName)
		if !promptConfirmation(message) {
			log.Info("Desinstalação cancelada pelo usuário", "release", releaseName)
			return nil
		}
	}

	// 2. Executar a desinstalação
	log.Info("Executando 'nelm release uninstall'...", "release", releaseName)

	uninstallArgs := []string{"release", "uninstall", "--kube-context", kubeContext, "-r", releaseName, "-n", releaseName}

	if err := runNelmCmdWithOutput(uninstallArgs, releaseDir, timeout); err != nil {
		return fmt.Errorf("falha na desinstalação: %w", err)
	}

	log.Info("Desinstalação concluída com sucesso!", "release", releaseName)
	return nil
}

// ExecuteSmartUninstall orquestra a desinstalação de uma release específica
func ExecuteSmartUninstall(opts UninstallOptions) error {
	// Validação das opções obrigatórias
	if opts.ReleaseName == "" {
		return fmt.Errorf("nome da release (ReleaseName) é obrigatório")
	}
	if opts.KubeContext == "" {
		return fmt.Errorf("contexto do Kubernetes (KubeContext) é obrigatório")
	}

	// Encontrar o diretório da release
	dirs, err := findReleaseDirs(opts.ReleaseName)
	if err != nil {
		return err
	}

	// Executar nelm para esta release
	return runNelmForUninstall(dirs[0], opts.ReleaseName, opts.KubeContext, opts.AutoApprove, opts.Timeout)
}
