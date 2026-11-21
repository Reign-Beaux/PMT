package options

import (
	"fmt"

	"github.com/spf13/viper"
)

// AuthSettings indica si la cookie de sesión exige HTTPS.
type AuthSettings struct {
	CookieSecure bool `mapstructure:"cookie_secure"`
}

func LoadAuthSettingsFrom(v *viper.Viper) (*AuthSettings, error) {
	var s AuthSettings
	if err := v.UnmarshalKey("auth", &s); err != nil {
		return nil, fmt.Errorf("auth.cookie_secure: configuración inválida")
	}
	return &s, nil
}
