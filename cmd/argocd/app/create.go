// cmd/argocd/app/create.go
package app

import (
	"context"
	"time"

	"github.com/estudosdevops/opsmaster/internal/argocd"
	"github.com/estudosdevops/opsmaster/internal/logger"
	"github.com/spf13/cobra"
)

var appCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Cria ou atualiza uma aplicação no Argo CD a partir de um repositório",
	Long:  `Cria uma Application no Argo CD apontando para um repositório Git que contém um Helm Chart e um arquivo de valores.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		log := logger.Get()
		serverAddr, _ := cmd.Flags().GetString("server")
		authToken, _ := cmd.Flags().GetString("token")
		insecure, _ := cmd.Flags().GetBool("insecure")

		appOpts := argocd.AppOptions{
			AppName:        cmd.Flag("app-name").Value.String(),
			Project:        cmd.Flag("project").Value.String(),
			DestinationNS:  cmd.Flag("dest-namespace").Value.String(),
			RepoURL:        cmd.Flag("repo-url").Value.String(),
			RepoPath:       cmd.Flag("repo-path").Value.String(),
			TargetRevision: cmd.Flag("target-revision").Value.String(),
			ValuesFile:     cmd.Flag("values").Value.String(),
			ImageRepo:      cmd.Flag("set-image-repo").Value.String(),
			ImageTag:       cmd.Flag("set-image-tag").Value.String(),
			DependencyName: cmd.Flag("set-chart-dependency").Value.String(),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		log.Info("Criando/atualizando aplicação no Argo CD...", "aplicação", appOpts.AppName)
		if err := argocd.CreateApplication(ctx, serverAddr, authToken, insecure, appOpts); err != nil {
			log.Error("Falha ao criar a aplicação", "erro", err)
			return err
		}
		log.Info("Aplicação criada/atualizada com sucesso!", "aplicação", appOpts.AppName)
		return nil
	},
}

func init() {
	appCreateCmd.Flags().String("app-name", "", "Nome da aplicação no Argo CD (obrigatório)")
	appCreateCmd.Flags().String("project", "default", "Projeto do Argo CD ao qual a aplicação pertencerá")
	appCreateCmd.Flags().String("dest-namespace", "", "Namespace de destino no Kubernetes (obrigatório)")
	appCreateCmd.Flags().String("repo-url", "", "URL do repositório Git que contém o chart (obrigatório)")
	appCreateCmd.Flags().String("repo-path", ".", "Caminho para o diretório do chart dentro do repositório")
	appCreateCmd.Flags().String("target-revision", "HEAD", "Branch, tag ou commit do Git a ser usado")
	appCreateCmd.Flags().String("values", "values.yaml", "Caminho para o arquivo de valores dentro do repositório (relativo ao repo-path)")
	appCreateCmd.Flags().String("set-image-repo", "", "Define o nome do repositório da imagem (obrigatório)")
	appCreateCmd.Flags().String("set-image-tag", "", "Define a tag da imagem a ser usada no deploy (obrigatório)")
	appCreateCmd.Flags().String("set-chart-dependency", "generic-app", "Nome da dependência do chart no Chart.yaml")

	appCreateCmd.MarkFlagRequired("app-name")
	appCreateCmd.MarkFlagRequired("dest-namespace")
	appCreateCmd.MarkFlagRequired("repo-url")
	appCreateCmd.MarkFlagRequired("set-image-repo")
	appCreateCmd.MarkFlagRequired("set-image-tag")
}
