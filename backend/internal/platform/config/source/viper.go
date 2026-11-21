// Package source arma el Viper que lee la configuración del ambiente.
package source

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// NewViper apunta a CONFIG_FILE o, si falta, a config/config.<ENV>.yaml
// (ENV por omisión dev). Las variables de entorno sobrescriben cada llave con
// su nombre en mayúsculas y "." cambiado por "_".
func NewViper() *viper.Viper {
	path := os.Getenv("CONFIG_FILE")
	if path == "" {
		env := os.Getenv("ENV")
		if env == "" {
			env = "dev"
		}
		path = filepath.Join("config", "config."+env+".yaml")
	}
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	return v
}
