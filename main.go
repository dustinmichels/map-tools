package main

import (
	"embed"
	"io/fs"
	"log/slog"
	"os"

	"github.com/dustinmichels/map-tools/internal/server"
)

//go:embed all:web/dist
var webFS embed.FS

func main() {
	distFS, err := fs.Sub(webFS, "web/dist")
	if err != nil {
		slog.Error("failed to sub web FS", "err", err)
		os.Exit(1)
	}

	addr := envOr("ADDR", ":8080")
	srv := server.New(distFS)

	slog.Info("server starting", "addr", addr)
	if err := srv.ListenAndServe(addr); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
