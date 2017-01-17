package config

import (
	"testing"
)

func TestLoadFromYAML(t *testing.T) {
	yaml := `ID: foo`
	config, err := LoadFromYAML([]byte(yaml))
	if err != nil {
		t.Fatal(err)
	}

	if config.ID != "foo" {
		t.Errorf("ID is %q, want %q", config.ID, "foo")
	}
}
