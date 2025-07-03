// cmd/get/ip.go
package get

import (
	"fmt"

	"github.com/estudosdevops/opsmaster/internal/ip"
	"github.com/estudosdevops/opsmaster/internal/logger"
	"github.com/estudosdevops/opsmaster/internal/presenter"

	"github.com/spf13/cobra"
)

var (
	showLocal  bool
	showPublic bool
)

var ipCmd = &cobra.Command{
	Use:   "ip",
	Short: "Busca e exibe seus endereços de IP (público e/ou local)",
	Long:  `Busca seu endereço de IP público na internet e/ou seu endereço de IP local na rede.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		log := logger.Get()
		showAll := !showLocal && !showPublic

		if showLocal || showAll {
			log.Info("Buscando informações da rede local...")
			localInfo, err := ip.FetchLocalNetworkInfo()
			if err != nil {
				log.Error("Não foi possível obter as informações da rede local", "erro", err)
			} else {
				header := []string{"TIPO", "INTERFACE", "IP", "MÁSCARA", "GATEWAY", "MAC ADDRESS"}
				row := []string{"Local", localInfo.InterfaceName, localInfo.IPAddress, localInfo.SubnetMask, localInfo.Gateway, localInfo.MACAddress}
				presenter.PrintTable(header, [][]string{row})
			}
			if showAll {
				fmt.Println()
			}
		}

		if showPublic || showAll {
			log.Info("Buscando informações do IP público...")
			publicInfo, err := ip.FetchPublicIP()
			if err != nil {
				log.Error("Não foi possível obter o IP público", "erro", err)
			} else {
				header := []string{"TIPO", "IP", "CIDADE", "REGIÃO", "PAÍS", "ORGANIZAÇÃO"}
				row := []string{"Público", publicInfo.IP, publicInfo.City, publicInfo.Region, publicInfo.Country, publicInfo.Org}
				presenter.PrintTable(header, [][]string{row})
			}
		}
		return nil
	},
}

func init() {
	// As flags são definidas aqui, mas o comando é adicionado ao pai no arquivo get.go.
	ipCmd.Flags().BoolVarP(&showLocal, "local", "l", false, "Exibe apenas o endereço de IP local")
	ipCmd.Flags().BoolVarP(&showPublic, "public", "p", false, "Exibe apenas o endereço de IP público")
}
