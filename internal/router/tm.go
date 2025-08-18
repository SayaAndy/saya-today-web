package router

import (
	"log/slog"
	"os"

	"github.com/SayaAndy/saya-today-web/internal/templatemanager"
)

var tm = assert(templatemanager.NewTemplateManager())

func assert[T any](t T, err error) T {
	if err != nil {
		slog.Error("fail to initialize template manager", slog.String("error", err.Error()))
		os.Exit(1)
	}
	return t
}
