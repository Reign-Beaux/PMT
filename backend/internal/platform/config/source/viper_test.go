package source

import "testing"

func TestConfigPath(t *testing.T) {
	t.Setenv("CONFIG_FILE", "")
	t.Setenv("ENV", "")
	v := NewViper()
	_ = v.ReadInConfig()
	if v.ConfigFileUsed() != "config/config.dev.yaml" {
		t.Fatalf("ruta: %s", v.ConfigFileUsed())
	}
	t.Setenv("ENV", "prod")
	v = NewViper()
	_ = v.ReadInConfig()
	if v.ConfigFileUsed() != "config/config.prod.yaml" {
		t.Fatalf("ruta: %s", v.ConfigFileUsed())
	}
	t.Setenv("CONFIG_FILE", "custom.yaml")
	v = NewViper()
	_ = v.ReadInConfig()
	if v.ConfigFileUsed() != "custom.yaml" {
		t.Fatalf("ruta: %s", v.ConfigFileUsed())
	}
}
