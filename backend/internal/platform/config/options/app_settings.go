package options

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// AppSettings tiene la URL pública de PMT: la base de los enlaces de invitación.
type AppSettings struct {
	PublicURL string `mapstructure:"public_url"`
}

func LoadAppSettingsFrom(v *viper.Viper) (*AppSettings, error) {
	var s AppSettings
	if err := v.UnmarshalKey("app", &s); err != nil {
		return nil, fmt.Errorf("app: configuración inválida")
	}
	if err := s.validate(); err != nil {
		return nil, err
	}
	return &s, nil
}

func (s *AppSettings) validate() error {
	if strings.TrimSpace(s.PublicURL) == "" {
		return fmt.Errorf("app.public_url es obligatoria")
	}
	return nil
}
