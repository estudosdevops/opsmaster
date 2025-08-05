// internal/nelm/types.go
package nelm

import "time"

// BaseOptions contém os campos comuns a todos os commandos nelm
type BaseOptions struct {
	// Campos obrigatórios
	KubeContext string // Contexto do Kubernetes a ser usado

	// Campos opcionais
	ReleaseName string        // Nome da release específica
	Namespace   string        // Namespace onde a release será processada
	AutoApprove bool          // Pula confirmação interativa
	Timeout     time.Duration // Timeout para commandos nelm (padrão: 5 minutos)
}

// InstallOptions armazena todas as opções para os commandos 'install'
type InstallOptions struct {
	BaseOptions
	// Campos específicos do install
	Environment    string // Ambiente de destino (ex: stg, prd) (obrigatório)
	MaxConcurrency int    // Máximo de releases executadas em paralelo (padrão: 3)
}

// UninstallOptions armazena todas as opções para o commando uninstall
type UninstallOptions struct {
	BaseOptions
	// ReleaseName é obrigatório para uninstall
}

// StatusOptions armazena todas as opções para o commando status
type StatusOptions struct {
	BaseOptions
	// ReleaseName é opcional para status (se vazio, lista todas)
}

// RollbackOptions armazena todas as opções para o commando rollback
type RollbackOptions struct {
	BaseOptions
	// Campos específicos do rollback
	Revision int // Revisão para fazer rollback (se 0, usa a anterior)
}

// Métodos helpers para acessar campos da base
func (b *BaseOptions) GetKubeContext() string {
	return b.KubeContext
}

func (b *BaseOptions) GetReleaseName() string {
	return b.ReleaseName
}

func (b *BaseOptions) GetNamespace() string {
	return b.Namespace
}

func (b *BaseOptions) GetAutoApprove() bool {
	return b.AutoApprove
}

func (b *BaseOptions) GetTimeout() time.Duration {
	return b.Timeout
}
