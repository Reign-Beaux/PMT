// Package config carga la configuración de la aplicación una sola vez al arrancar.
package config

import (
	"fmt"
	"path/filepath"

	"pmt/internal/platform/config/options"
	"pmt/internal/platform/config/source"
)

// Config contiene la configuración tipada por sección.
type Config struct {
	App      *options.AppSettings
	Server   *options.ServerSettings
	Database *options.DatabaseSettings
	Frontend *options.FrontendSettings
	CLI      *options.CLISettings
	Auth     *options.AuthSettings
	Owner    *options.OwnerSettings
}

// Load lee el archivo del ambiente, aplica las variables de entorno y valida
// cada sección. Un frontend.dist o un cli.dist relativos se resuelven contra el
// directorio del archivo de configuración.
func Load() (Config, error) {
	v := source.NewViper()
	if err := v.ReadInConfig(); err != nil {
		return Config{}, fmt.Errorf("leer configuración %s: %w", v.ConfigFileUsed(), err)
	}
	for _, key := range v.AllKeys() {
		if err := v.BindEnv(key); err != nil {
			return Config{}, fmt.Errorf("enlazar %s: %w", key, err)
		}
		v.Set(key, v.Get(key))
	}
	for _, key := range []string{"frontend.dist", "cli.dist"} {
		if dist := v.GetString(key); dist != "" && !filepath.IsAbs(dist) {
			abs, err := filepath.Abs(filepath.Join(filepath.Dir(v.ConfigFileUsed()), dist))
			if err != nil {
				return Config{}, fmt.Errorf("%s: %w", key, err)
			}
			v.Set(key, abs)
		}
	}
	var c Config
	var err error
	if c.App, err = options.LoadAppSettingsFrom(v); err != nil {
		return Config{}, err
	}
	if c.Server, err = options.LoadServerSettingsFrom(v); err != nil {
		return Config{}, err
	}
	if c.Database, err = options.LoadDatabaseSettingsFrom(v); err != nil {
		return Config{}, err
	}
	if c.Frontend, err = options.LoadFrontendSettingsFrom(v); err != nil {
		return Config{}, err
	}
	if c.CLI, err = options.LoadCLISettingsFrom(v); err != nil {
		return Config{}, err
	}
	if c.Auth, err = options.LoadAuthSettingsFrom(v); err != nil {
		return Config{}, err
	}
	if c.Owner, err = options.LoadOwnerSettingsFrom(v); err != nil {
		return Config{}, err
	}
	return c, nil
}
