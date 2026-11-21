// Package options define cada sección de la configuración y su validación.
package options

import (
	"fmt"
	"github.com/spf13/viper"
	"strings"
)

type ServerSettings struct {
	Addr string `mapstructure:"addr"`
}

func LoadServerSettingsFrom(v *viper.Viper) (*ServerSettings, error) {
	var s ServerSettings
	if err := v.UnmarshalKey("server", &s); err != nil {
		return nil, fmt.Errorf("server: configuración inválida")
	}
	if err := s.validate(); err != nil {
		return nil, err
	}
	return &s, nil
}

func (s *ServerSettings) validate() error {
	if strings.TrimSpace(s.Addr) == "" {
		return fmt.Errorf("server.addr es obligatoria")
	}
	return nil
}
