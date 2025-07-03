// cmd/scan/ports.go
package scan

import (
	"net"
	"sort"
	"strconv"
	"time"

	"github.com/estudosdevops/opsmaster/internal/logger"
	"github.com/estudosdevops/opsmaster/internal/presenter"
	"github.com/estudosdevops/opsmaster/internal/scanner"

	"github.com/spf13/cobra"
)

var (
	portRange string
	timeout   time.Duration
)

var portsCmd = &cobra.Command{
	Use:   "ports <host>",
	Short: "Escaneia portas TCP em um host",
	Long:  `Verifica o status de portas TCP (abertas, fechadas ou filtradas) em um determinado host.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		log := logger.Get()
		host := args[0]

		log.Info("Validando o host...", "host", host)
		if _, err := net.LookupHost(host); err != nil {
			log.Error("Host inválido ou não foi possível resolver o nome", "host", host, "erro", err)
			return err
		}

		log.Info("Iniciando escaneamento de portas...", "host", host, "portas", portRange, "timeout", timeout)

		results, err := scanner.ScanPorts(host, portRange, timeout)
		if err != nil {
			log.Error("Erro ao preparar o escaneamento", "erro", err)
			return err
		}

		if len(results) == 0 {
			log.Info("Nenhuma porta foi escaneada ou todas as portas estão fechadas/filtradas.")
			return nil
		}

		sort.Slice(results, func(i, j int) bool {
			return results[i].Port < results[j].Port
		})

		header := []string{"PORTA", "STATUS"}
		var rows [][]string
		for _, res := range results {
			rows = append(rows, []string{strconv.Itoa(res.Port), res.Status})
		}

		presenter.PrintTable(header, rows)
		return nil
	},
}

func init() {
	// A função init() deste arquivo apenas define as flags do comando 'ports'.
	portsCmd.Flags().StringVarP(&portRange, "ports", "p", "1-1024", "Portas a escanear (ex: 80,443 ou 1-1024)")
	portsCmd.Flags().DurationVarP(&timeout, "timeout", "t", 2*time.Second, "Timeout para cada conexão de porta (ex: 500ms, 2s)")
}
