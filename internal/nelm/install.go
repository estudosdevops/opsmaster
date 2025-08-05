// internal/nelm/install.go
package nelm

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/estudosdevops/opsmaster/internal/logger"
	"github.com/estudosdevops/opsmaster/internal/presenter"
)

// runNelmForRelease executa os commandos nelm para uma release específica
func runNelmForRelease(releaseDir, releaseName, env, kubeContext string, autoApprove bool, timeout time.Duration) error {
	log := logger.Get()

	// Usar timeout padrão se não fornecido
	timeout = getDefaultTimeout(timeout)

	// Baixar e atualizar dependências do chart
	log.Info("Baixando e atualizando dependências do chart", "release", releaseName)
	dependencyArgs := []string{"chart", "dependency", "update"}
	if err := runNelmCmdWithOutput(dependencyArgs, releaseDir, timeout); err != nil {
		return fmt.Errorf("falha ao baixar/atualizar dependências do chart: %w", err)
	}
	log.Info("Dependências do chart atualizadas com sucesso!", "release", releaseName)

	log.Info("Validações prévias concluídas", "release", releaseName)

	// 1. Validar o chart
	log.Info("Executando 'nelm chart lint'", "release", releaseName)

	lintArgs := []string{"chart", "lint"}
	if err := runNelmCmdWithOutput(lintArgs, releaseDir, timeout); err != nil {
		return fmt.Errorf("falha na validação do chart: %w", err)
	}

	log.Info("Validação do chart concluída com sucesso!", "release", releaseName)

	// 2. Planejar a instalação
	log.Info("Executando 'nelm release plan install'...", "release", releaseName)

	// Verificar se existe arquivo de valores específico do ambiente
	envValuesFile := fmt.Sprintf("values-%s.yaml", env)
	envValuesPath := filepath.Join(releaseDir, envValuesFile)

	planArgs := []string{"release", "plan", "install", "--kube-context", kubeContext, "-r", releaseName, "-n", releaseName}

	if _, err := os.Stat(envValuesPath); err == nil {
		// Se existe arquivo de valores específico, usar ele
		log.Info("Usando arquivo de valores específico do ambiente", "file", envValuesFile, "release", releaseName)
		planArgs = append(planArgs, "--values", envValuesFile)
	} else {
		// Se não existe, usar --set environment
		log.Info("Usando --set environment", "env", env, "release", releaseName)
		planArgs = append(planArgs, "--set", fmt.Sprintf("environment=%s", env))
	}

	if err := runNelmCmdWithOutput(planArgs, releaseDir, timeout); err != nil {
		return fmt.Errorf("falha no planejamento da instalação: %w", err)
	}

	// 3. Se não for auto-approve, perguntar ao usuário
	if !autoApprove {
		// Determinar valores
		valuesStr := envValuesFile
		if _, err := os.Stat(envValuesPath); err != nil {
			valuesStr = fmt.Sprintf("--set environment=%s", env)
		}

		// Criar tabela horizontal
		header := []string{"Release", "Ambiente", "Namespace", "KubeCtx", "Valores"}
		rows := [][]string{
			{releaseName, strings.ToUpper(env), releaseName, kubeContext, valuesStr},
		}

		fmt.Printf("\nResumo da instalação para '%s':\n", releaseName)
		presenter.PrintTable(header, rows)
		fmt.Println()

		if !promptConfirmation("Deseja aplicar estas alterações? (s/N): ") {
			log.Info("Instalação cancelada pelo usuário", "release", releaseName)
			return nil // Retorna imediatamente, sem executar mais nada
		}
	}

	// 4. Executar a instalação
	log.Info("Executando 'nelm release install'...", "release", releaseName)

	installArgs := []string{"release", "install", "--kube-context", kubeContext, "-r", releaseName, "-n", releaseName}

	if _, err := os.Stat(envValuesPath); err == nil {
		installArgs = append(installArgs, "--values", envValuesFile)
	} else {
		installArgs = append(installArgs, "--set", fmt.Sprintf("environment=%s", env))
	}

	if err := runNelmCmdWithOutput(installArgs, releaseDir, timeout); err != nil {
		return fmt.Errorf("falha na instalação: %w", err)
	}

	log.Info("Instalação concluída com sucesso!", "release", releaseName)
	return nil
}

// ExecuteSmartInstall orquestra o fluxo para uma ou todas as releases
func ExecuteSmartInstall(opts *InstallOptions) error {
	log := logger.Get()

	// Validação das opções obrigatórias
	if err := validateRequiredFields(opts.Environment, opts.KubeContext); err != nil {
		return err
	}

	// Se uma release específica foi fornecida
	if opts.ReleaseName != "" {
		dirs, err := findReleaseDirs(opts.ReleaseName)
		if err != nil {
			return err
		}
		if len(dirs) == 0 {
			return fmt.Errorf("release '%s' não encontrada no repositório", opts.ReleaseName)
		}

		log.Info("Processando release específica", "release", opts.ReleaseName)

		// Executar nelm para esta release específica
		if err := runNelmForRelease(dirs[0], opts.ReleaseName, opts.Environment, opts.KubeContext, opts.AutoApprove, opts.Timeout); err != nil {
			log.Error("Falha ao processar release", "release", opts.ReleaseName, "erro", err)
			return err
		}

		// Retornar após processar a release específica
		return nil
	}

	// Se nenhuma release específica foi fornecida, processar todas as releases
	dirs, err := findReleaseDirs("") // Passando "" para buscar todas as releases
	if err != nil {
		return err
	}
	if len(dirs) == 0 {
		log.Info("Nenhuma release (Chart.yaml) encontrada no repositório.")
		return nil
	}

	log.Info("Encontradas releases para processamento", "total", len(dirs))
	for i, dir := range dirs {
		log.Info("Release encontrada", "index", i+1, "name", filepath.Base(dir), "path", dir)
	}

	// Executar releases em paralelo
	return executeReleasesInParallel(dirs, opts)
}

// executeReleasesInParallel executa múltiplas releases em paralelo
func executeReleasesInParallel(dirs []string, opts *InstallOptions) error {
	log := logger.Get()

	// Limitar concorrência para evitar sobrecarregar o sistema
	maxConcurrency := opts.MaxConcurrency
	if maxConcurrency <= 0 {
		maxConcurrency = 3 // padrão: 3 releases simultâneas
	}

	semaphore := make(chan struct{}, maxConcurrency)

	// Channel para coletar resultados
	results := make(chan error, len(dirs))

	log.Info("Executando releases em paralelo", "total", len(dirs), "maxConcurrency", maxConcurrency)

	// Iniciar goroutines para cada release
	for _, dir := range dirs {
		releaseName := filepath.Base(dir)
		namespace := opts.Namespace
		if namespace == "" {
			namespace = releaseName
		}

		go func(releaseDir, releaseName, _ string) { // Corrigir unused-parameter
			// Adquirir semáforo para limitar concorrência
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			log.Info("Iniciando processamento da release", "release", releaseName)
			err := runNelmForRelease(releaseDir, releaseName, opts.Environment, opts.KubeContext, opts.AutoApprove, opts.Timeout)
			results <- err
		}(dir, releaseName, namespace)
	}

	// Coletar resultados
	erros := 0
	for range dirs {
		if err := <-results; err != nil {
			log.Error("Falha ao instalar a release", "erro", err)
			erros++
		}
	}

	if erros > 0 {
		return fmt.Errorf("%d releases falharam na instalação", erros)
	}

	log.Info("Todas as releases foram processadas com sucesso", "total", len(dirs))
	return nil
}
