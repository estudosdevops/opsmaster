// cmd/get/dns.go
package get

import (
	"strings"

	"github.com/estudosdevops/opsmaster/internal/dns"
	"github.com/estudosdevops/opsmaster/internal/logger"
	"github.com/estudosdevops/opsmaster/internal/presenter"

	"github.com/spf13/cobra"
)

var recordType string

var dnsCmd = &cobra.Command{
	Use:   "dns <dominio>",
	Short: "Busca registros DNS de um domínio (similar a 'dig')",
	Long:  `Realiza uma consulta DNS para encontrar registros de um tipo específico (A, AAAA, MX, TXT, etc.) associados a um nome de domínio.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		log := logger.Get()
		domain := args[0]
		queryType := strings.ToUpper(recordType)

		log.Info("Consultando registros DNS...", "domínio", domain, "tipo", queryType)

		records, err := dns.Query(domain, queryType)
		if err != nil {
			log.Error("Falha na consulta DNS", "erro", err)
			return err
		}

		if len(records) == 0 {
			log.Info("Nenhum registro encontrado para este domínio com o tipo especificado.")
			return nil
		}

		header := []string{"DOMÍNIO", "TIPO", "VALOR"}
		var rows [][]string
		for _, record := range records {
			rows = append(rows, []string{domain, queryType, record})
		}
		presenter.PrintTable(header, rows)
		return nil
	},
}

func init() {
	dnsCmd.Flags().StringVarP(&recordType, "type", "t", "A", "Tipo de registro DNS a consultar (ex: A, AAAA, MX, TXT, CNAME, NS)")
}
