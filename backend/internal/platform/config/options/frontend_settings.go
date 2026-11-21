package options

import (
	"fmt"

	"github.com/spf13/viper"
)

// FrontendSettings indica dónde está el compilado del frontend. Si dist no
// existe o no tiene index.html el servidor arranca igual: platform/web registra
// un aviso y las rutas de la interfaz responden "frontend no compilado".
type FrontendSettings struct {
	Dist string `mapstructure:"dist"`
}

func LoadFrontendSettingsFrom(v *viper.Viper) (*FrontendSettings, error) {
	var s FrontendSettings
	if err := v.UnmarshalKey("frontend", &s); err != nil {
		return nil, fmt.Errorf("frontend: configuración inválida")
	}
	return &s, nil
}
