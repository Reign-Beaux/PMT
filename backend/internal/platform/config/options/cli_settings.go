package options

import (
	"fmt"

	"github.com/spf13/viper"
)

// CLISettings indica dónde están los binarios del CLI pmt que sirve el servidor
// en /descargas y /install.sh. Si dist está vacío, no existe o le faltan
// binarios, el servidor arranca igual: esas rutas responden que no hay binarios.
type CLISettings struct {
	Dist string `mapstructure:"dist"`
}

func LoadCLISettingsFrom(v *viper.Viper) (*CLISettings, error) {
	var s CLISettings
	if err := v.UnmarshalKey("cli", &s); err != nil {
		return nil, fmt.Errorf("cli: configuración inválida")
	}
	return &s, nil
}
