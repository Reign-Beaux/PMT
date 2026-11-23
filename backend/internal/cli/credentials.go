package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Credentials es la sesión del CLI: el servidor y el token de API del usuario.
// Vive en ~/.config/pmt/credenciales.yaml con permiso 600 y la carpeta con 700.
type Credentials struct {
	Server   string `yaml:"servidor"`
	Username string `yaml:"username"`
	Token    string `yaml:"token"`
	TokenID  string `yaml:"token_id"`
}

// errNoCredentials indica que no hay sesión guardada.
var errNoCredentials = errors.New("no has iniciado sesión; corre «pmt login»")

// configDir es ~/.config/pmt. No usa ~/.config/pmt/tokens/, que es del kit.
func configDir(home string) string {
	return filepath.Join(home, ".config", "pmt")
}

// credentialsPath es la ruta del archivo de credenciales.
func credentialsPath(home string) string {
	return filepath.Join(configDir(home), "credenciales.yaml")
}

// loadCredentials lee las credenciales. Sin archivo devuelve errNoCredentials.
func loadCredentials(home string) (Credentials, error) {
	raw, err := os.ReadFile(credentialsPath(home))
	if errors.Is(err, fs.ErrNotExist) {
		return Credentials{}, errNoCredentials
	}
	if err != nil {
		return Credentials{}, fmt.Errorf("leer credenciales: %w", err)
	}
	var c Credentials
	if err := yaml.Unmarshal(raw, &c); err != nil {
		return Credentials{}, fmt.Errorf("credenciales dañadas en %s: %w", credentialsPath(home), err)
	}
	if c.Server == "" || c.Token == "" {
		return Credentials{}, errNoCredentials
	}
	return c, nil
}

// saveCredentials escribe las credenciales con permiso 600 en una carpeta 700.
// Escribe a un archivo temporal y lo renombra, para no dejar uno a medias.
func saveCredentials(home string, c Credentials) error {
	dir := configDir(home)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("crear %s: %w", dir, err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return fmt.Errorf("permisos de %s: %w", dir, err)
	}
	raw, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".credenciales-*")
	if err != nil {
		return fmt.Errorf("guardar credenciales: %w", err)
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return fmt.Errorf("guardar credenciales: %w", err)
	}
	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		return fmt.Errorf("guardar credenciales: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("guardar credenciales: %w", err)
	}
	if err := os.Rename(tmp.Name(), credentialsPath(home)); err != nil {
		return fmt.Errorf("guardar credenciales: %w", err)
	}
	return nil
}

// removeCredentials borra el archivo de credenciales, si existe.
func removeCredentials(home string) error {
	err := os.Remove(credentialsPath(home))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("borrar credenciales: %w", err)
	}
	return nil
}
