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
	Server   *options.ServerSettings
	Database *options.DatabaseSettings
	Frontend *options.FrontendSettings
}

// Load lee el archivo del ambiente, aplica las variables de entorno y valida
// cada sección. Un frontend.dist relativo se resuelve contra el directorio del
// archivo de configuración.
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
	if dist := v.GetString("frontend.dist"); dist != "" && !filepath.IsAbs(dist) {
		abs, err := filepath.Abs(filepath.Join(filepath.Dir(v.ConfigFileUsed()), dist))
		if err != nil {
			return Config{}, fmt.Errorf("frontend.dist: %w", err)
		}
		v.Set("frontend.dist", abs)
	}
	var c Config
	var err error
	if c.Server, err = options.LoadServerSettingsFrom(v); err != nil {
		return Config{}, err
	}
	if c.Database, err = options.LoadDatabaseSettingsFrom(v); err != nil {
		return Config{}, err
	}
	if c.Frontend, err = options.LoadFrontendSettingsFrom(v); err != nil {
		return Config{}, err
	}
	return c, nil
}
