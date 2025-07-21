package presenter

import (
	"fmt"
	"os"
	"text/tabwriter"
)

// PrintTable formata e imprime dados em uma tabela no console.
// Ela recebe o cabeçalho e as linhas como fatias de strings.
func PrintTable(header []string, rows [][]string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)

	// Imprime o cabeçalho
	headerStr := ""
	separatorStr := ""
	for i, col := range header {
		headerStr += col
		// Cria a linha separadro (ex: "----- ) com o mesmo tamanho do cabeçalho
		for range len(col) {
			separatorStr += "-"
		}
		if i < len(header)-1 {
			headerStr += "\t"
			separatorStr += "\t"
		}
	}
	fmt.Fprintln(w, headerStr)
	fmt.Fprintln(w, separatorStr)

	// Imprime as linhas de dados
	for _, row := range rows {
		rowStr := ""
		for i, cell := range row {
			rowStr += cell
			if i < len(row)-1 {
				rowStr += "\t"
			}
		}
		fmt.Fprintf(w, "%s\n", rowStr)
	}
	w.Flush()
}
