// cmd/scan/monitor.go
package scan

import (
	"time"

	"github.com/estudosdevops/opsmaster/internal/monitor"
	"github.com/spf13/cobra"
)

var (
	monitorInterval time.Duration
	monitorCount    int
)

var monitorCmd = &cobra.Command{
	Use:   "monitor <url>",
	Short: "Monitora uma URL em intervalos regulares",
	Long:  `Realiza requisições HTTP para uma URL em intervalos de tempo definidos para verificar a sua disponibilidade e status.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		url := args[0]
		// CORREÇÃO: A chamada da função agora está correta, sem o parâmetro de contexto.
		monitor.StartMonitoring(url, monitorInterval, monitorCount)
		return nil
	},
}

func init() {
	// A função init() deste arquivo apenas define as flags do comando 'monitor'.
	monitorCmd.Flags().DurationVarP(&monitorInterval, "interval", "i", 10*time.Second, "Intervalo entre as verificações")
	monitorCmd.Flags().IntVarP(&monitorCount, "count", "c", 0, "Número de verificações a serem feitas (0 para infinito)")
}
