package options

import (
	"fmt"
	"github.com/spf13/viper"
	"strings"
)

type DatabaseSettings struct {
	ConnectionString string `mapstructure:"connection_string"`
}

func LoadDatabaseSettingsFrom(v *viper.Viper) (*DatabaseSettings, error) {
	var s DatabaseSettings
	if err := v.UnmarshalKey("database", &s); err != nil {
		return nil, fmt.Errorf("database: configuración inválida")
	}
	if err := s.validate(); err != nil {
		return nil, err
	}
	return &s, nil
}

func (s *DatabaseSettings) validate() error {
	if strings.TrimSpace(s.ConnectionString) == "" {
		return fmt.Errorf("database.connection_string es obligatoria")
	}
	return nil
}
