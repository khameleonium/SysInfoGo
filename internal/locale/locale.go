package locale

import (
	"encoding/json"
	"os"
	"path/filepath"
)

var translations map[string]string

func Load() error {
	translations = make(map[string]string)

	exe, err := os.Executable()
	if err != nil {
		return err
	}
	dir := filepath.Dir(exe)
	path := filepath.Join(dir, "sysinfogo_locale.json")

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &translations)
}

func T(key string) string {
	if val, ok := translations[key]; ok {
		return val
	}
	return key
}

func GetDictionary() map[string]string {
	if translations == nil {
		return map[string]string{}
	}
	return translations
}

func SaveDefault(path string) error {
	data, err := json.MarshalIndent(DefaultDictionary, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
