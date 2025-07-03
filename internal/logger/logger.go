// opsmaster/internal/logger/logger.go
package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/fatih/color"
)

// CustomTextHandler é o nosso handler customizado que envolve o TextHandler padrão.
type CustomTextHandler struct {
	handler slog.Handler
}

// Get retorna uma instância pré-configurada do logger slog com um formato customizado.
func Get() *slog.Logger {
	handler := &CustomTextHandler{
		handler: slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			// Desativamos a adição automática de atributos para termos controle total.
			AddSource: false,
			// Removemos os atributos padrão de tempo e nível, pois vamos formatá-los manualmente.
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				if a.Key == slog.TimeKey || a.Key == slog.LevelKey || a.Key == slog.MessageKey {
					return slog.Attr{}
				}
				return a
			},
		}),
	}

	return slog.New(handler)
}

// Handle é o método que formata cada entrada de log.
func (h *CustomTextHandler) Handle(ctx context.Context, r slog.Record) error {
	// Formata o nível do log com cor.
	levelStr := r.Level.String()
	switch r.Level {
	case slog.LevelDebug:
		levelStr = color.MagentaString(levelStr)
	case slog.LevelInfo:
		levelStr = color.GreenString(levelStr)
	case slog.LevelWarn:
		levelStr = color.YellowString(levelStr)
	case slog.LevelError:
		levelStr = color.RedString(levelStr)
	}

	// Monta a string de atributos (chave=valor) que vêm depois da mensagem.
	attrs := ""
	r.Attrs(func(a slog.Attr) bool {
		attrs = attrs + " " + color.CyanString(a.Key+"=") + a.Value.String()
		return true
	})

	// Monta a linha de log final no formato desejado.
	// Ex: 15:04:05.000 [INFO] [opsmaster]: Mensagem principal key=value
	logLine := fmt.Sprintf("%s [%s] %s: %s%s\n",
		r.Time.Format("15:04:05.000"),
		levelStr,
		color.BlueString("[opsmaster]"), // Adiciona um prefixo para a nossa ferramenta
		r.Message,
		attrs,
	)

	// Escreve a linha formatada na saída padrão (console).
	_, err := os.Stdout.WriteString(logLine)
	return err
}

// WithAttrs e WithGroup são necessários para implementar a interface slog.Handler.
func (h *CustomTextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &CustomTextHandler{handler: h.handler.WithAttrs(attrs)}
}

func (h *CustomTextHandler) WithGroup(name string) slog.Handler {
	return &CustomTextHandler{handler: h.handler.WithGroup(name)}
}

func (h *CustomTextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}
