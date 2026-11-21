package options

import (
	"fmt"

	"github.com/spf13/viper"
)

// OwnerSettings son los datos del dueño de la instancia, que se crea al
// arrancar si no hay usuarios. Sin correo ni contraseña no se crea.
type OwnerSettings struct {
	Email    string `mapstructure:"email"`
	Password string `mapstructure:"password"`
	Username string `mapstructure:"username"`
}

func LoadOwnerSettingsFrom(v *viper.Viper) (*OwnerSettings, error) {
	var s OwnerSettings
	if err := v.UnmarshalKey("owner", &s); err != nil {
		return nil, fmt.Errorf("owner: configuración inválida")
	}
	return &s, nil
}
