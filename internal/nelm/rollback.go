// internal/nelm/rollback.go
package nelm

import (
	"fmt"
	"time"

	"github.com/estudosdevops/opsmaster/internal/logger"
)

// runNelmForRollback executa os commandos nelm para fazer rollback de uma release específica
func runNelmForRollback(releaseDir, releaseName, kubeContext string, revision int, autoApprove bool, timeout time.Duration) error {
	log := logger.Get()

	// Usar timeout padrão se não fornecido
	timeout = getDefaultTimeout(timeout)

	// 1. Se não for auto-approve, perguntar ao usuário
	if !autoApprove {
		message := fmt.Sprintf("Deseja fazer rollback da release '%s' para revisão %d? (s/N): ", releaseName, revision)
		if !promptConfirmation(message) {
			log.Info("Rollback cancelado pelo usuário", "release", releaseName)
			return nil
		}
	}

	// 2. Executar o rollback
	log.Info("Executando 'nelm release rollback'...", "release", releaseName, "revision", revision)

	// Sintaxe correta: nelm release rollback [options...] -n namespace -r release [revision]
	rollbackArgs := []string{"release", "rollback", "--kube-context", kubeContext, "-r", releaseName, "-n", releaseName}

	// Adicionar revisão como argumento positional (não como flag)
	if revision > 0 {
		rollbackArgs = append(rollbackArgs, fmt.Sprintf("%d", revision))
	}

	if err := runNelmCmdWithOutput(rollbackArgs, releaseDir, timeout); err != nil {
		return fmt.Errorf("falha no rollback: %w", err)
	}

	log.Info("Rollback concluído com sucesso!", "release", releaseName, "revision", revision)
	return nil
}

// ExecuteSmartRollback orquestra o rollback de uma release específica
func ExecuteSmartRollback(opts RollbackOptions) error {
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
	return runNelmForRollback(dirs[0], opts.ReleaseName, opts.KubeContext, opts.Revision, opts.AutoApprove, opts.Timeout)
}
