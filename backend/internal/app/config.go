package app

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/spf13/viper"
)

const (
	databaseURLKey      = "database.url"
	serverAddrKey       = "server.addr"
	frontendDistKey     = "frontend.dist"
	defaultFrontendDist = "../frontend/dist"
)

// Config contiene la configuración de la aplicación.
type Config struct {
	DatabaseURL  string
	ServerAddr   string
	FrontendDist string
}

// LoadConfig lee la configuración de path, que es opcional. Las variables de
// entorno DATABASE_URL, SERVER_ADDR y FRONTEND_DIST tienen prioridad sobre el
// archivo.
func LoadConfig(path string) (Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetDefault(serverAddrKey, "127.0.0.1:8080")
	v.SetDefault(frontendDistKey, defaultFrontendDist)
	if err := v.BindEnv(databaseURLKey, "DATABASE_URL"); err != nil {
		return Config{}, fmt.Errorf("enlazar DATABASE_URL: %w", err)
	}
	if err := v.BindEnv(serverAddrKey, "SERVER_ADDR"); err != nil {
		return Config{}, fmt.Errorf("enlazar SERVER_ADDR: %w", err)
	}
	if err := v.BindEnv(frontendDistKey, "FRONTEND_DIST"); err != nil {
		return Config{}, fmt.Errorf("enlazar FRONTEND_DIST: %w", err)
	}

	if err := v.ReadInConfig(); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Config{}, fmt.Errorf("leer %s: %w", path, err)
	}

	cfg := Config{
		DatabaseURL:  v.GetString(databaseURLKey),
		ServerAddr:   v.GetString(serverAddrKey),
		FrontendDist: v.GetString(frontendDistKey),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("falta database.url en la configuración")
	}
	if cfg.FrontendDist == "" {
		cfg.FrontendDist = defaultFrontendDist
	}
	return cfg, nil
}
