// internal/nelm/utils.go
package nelm

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/estudosdevops/opsmaster/internal/logger"
	"github.com/spf13/cobra"
)

// findReleaseDirs percorre recursivamente o repositório e retorna diretórios que possuem Chart.yaml
// Se releaseName for fornecido, retorna apenas o diretório daquela release
// Se releaseName for vazio, retorna todos os diretórios de release
func findReleaseDirs(releaseName string) ([]string, error) {
	var releases []string
	err := filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// Verificar se o diretório contém Chart.yaml
			chartPath := filepath.Join(path, "Chart.yaml")
			if _, err := os.Stat(chartPath); err == nil {
				dirName := filepath.Base(path)

				// Se releaseName foi especificado, filtrar apenas essa release
				if releaseName != "" {
					if dirName == releaseName {
						releases = append(releases, path)
					}
				} else {
					// Se não foi especificado, adicionar todos os diretórios com Chart.yaml
					releases = append(releases, path)
				}
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("erro ao percorrer o repositório: %w", err)
	}

	// Se releaseName foi especificado mas não foi encontrado
	if releaseName != "" && len(releases) == 0 {
		return nil, fmt.Errorf("release '%s' não encontrada no repositório", releaseName)
	}

	return releases, nil
}

// runNelmCmd executa um commando nelm no diretório especificado
// Se captureOutput for true, retorna a saída como string
// Se captureOutput for false, apenas executa o commando
func runNelmCmd(args []string, dir string, captureOutput bool, timeout time.Duration) (string, error) {
	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, "nelm", args...)
	cmd.Dir = dir

	if captureOutput {
		// Capturar saída para retornar (sem imprimir diretamente)
		var outputBuffer bytes.Buffer
		cmd.Stdout = &outputBuffer
		cmd.Stderr = &outputBuffer
		if err := cmd.Run(); err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				return outputBuffer.String(), fmt.Errorf("commando 'nelm %s' excedeu o timeout de %v", args[0], timeout)
			}
			return outputBuffer.String(), fmt.Errorf("falha ao executar o commando 'nelm %s': %w", args[0], err)
		}
		return outputBuffer.String(), nil
	}
	// Apenas executar sem capturar saída
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("commando 'nelm %s' excedeu o timeout de %v", args[0], timeout)
		}
		return "", err
	}
	return "", nil
}

// runNelmCmdWithOutput executa um commando nelm e sempre imprime a saída diretamente
// Útil para commandos como 'plan', 'install', 'uninstall' que precisam mostrar o output
func runNelmCmdWithOutput(args []string, dir string, timeout time.Duration) error {
	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, "nelm", args...)
	cmd.Dir = dir

	// Sempre imprimir diretamente
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("commando 'nelm %s' excedeu o timeout de %v", args[0], timeout)
		}
		return fmt.Errorf("falha ao executar o commando 'nelm %s': %w", args[0], err)
	}

	return nil
}

// promptConfirmation exibe uma pergunta e aguarda confirmação do usuário
func promptConfirmation(message string) bool {
	fmt.Print(message)
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	return strings.ToLower(strings.TrimSpace(response)) == "s"
}

const defaultTimeout = 5 * time.Minute

// getDefaultTimeout retorna o timeout padrão se não fornecido
func getDefaultTimeout(timeout time.Duration) time.Duration {
	if timeout == 0 {
		return defaultTimeout
	}
	return timeout
}

// validateRequiredFields valida campos obrigatórios
func validateRequiredFields(environment, kubeContext string) error {
	if environment == "" {
		return fmt.Errorf("ambiente (Environment) é obrigatório")
	}
	if kubeContext == "" {
		return fmt.Errorf("contexto do Kubernetes (KubeContext) é obrigatório")
	}
	return nil
}

// parseTimeoutFromFlag parseia uma string de timeout e retorna um time.Duration
// Se a string estiver vazia ou for inválida, retorna o valor padrão
func ParseTimeoutFromFlag(timeoutStr string, defaultTimeout time.Duration) time.Duration {
	if timeoutStr == "" {
		return defaultTimeout
	}
	if parsed, err := time.ParseDuration(timeoutStr); err == nil {
		return parsed
	}
	return defaultTimeout
}

// ReadCommonNelmFlags lê flags comuns dos commandos nelm
func ReadCommonNelmFlags(cmd *cobra.Command) (kubeContext, releaseName, namespace, timeoutStr string, autoApprove bool) {
	kubeContext, _ = cmd.Flags().GetString("kube-context")
	releaseName, _ = cmd.Flags().GetString("release")
	namespace, _ = cmd.Flags().GetString("namespace")
	timeoutStr, _ = cmd.Flags().GetString("timeout")
	autoApprove, _ = cmd.Flags().GetBool("auto-approve")
	return
}

// validateNelmRequiredFields valida campos obrigatórios específicos dos commandos nelm
func ValidateNelmRequiredFields(fields map[string]string) error {
	for field, value := range fields {
		if value == "" {
			return fmt.Errorf("campo '%s' é obrigatório", field)
		}
	}
	return nil
}

// LogNelmCommandStart loga o início de um commando nelm de forma padronizada
func LogNelmCommandStart(commandName string, fields map[string]any) {
	log := logger.Get()
	args := []any{}
	for key, value := range fields {
		args = append(args, key, value)
	}
	log.Info(fmt.Sprintf("Iniciando o fluxo de %s do nelm...", commandName), args...)
}

// LogNelmCommandSuccess loga o sucesso de um commando nelm de forma padronizada
func LogNelmCommandSuccess(commandName string) {
	log := logger.Get()
	log.Info(fmt.Sprintf("Fluxo de %s concluído com sucesso!", commandName))
}

// LogNelmCommandError loga o erro de um commando nelm de forma padronizada
func LogNelmCommandError(commandName string, err error) {
	log := logger.Get()
	log.Error(fmt.Sprintf("O fluxo de %s falhou", commandName), "erro", err)
}
