package utils

import (
	"bytes"
	"debug/elf"
	"os"
	"path/filepath"
	"runtime"
	"text/template"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var funcs = template.FuncMap{
	// Eden config parameter
	"EdenConfig": func(key string) string {
		val := viper.GetString(key)
		return val
	},
	// Resolve path from Eden config parameter relative
	// to the Eden root directory
	"EdenConfigPath": func(path string) string {
		res := ResolveAbsPath(viper.GetString(path))
		return res
	},
	// Resolve path relative to the Eden root directory
	"EdenPath": func(path string) string {
		res := ResolveAbsPath(path)
		return res
	},
	// Retrieves the value of the environment variable named by the key.
	"EdenGetEnv": func(key string) string {
		res := os.Getenv(key)
		return res
	},
	// Get the runtime Operating system name
	"EdenOSRuntime": func() string {
		return runtime.GOOS
	},
	// Check libslirp version. Version 4.2 and later do not support communication between slirp interfaces
	"EdenCheckSlirpSupportRouting": func() bool {
		pathToSearch := ""
		if err := filepath.Walk("/usr/lib", func(path string, info os.FileInfo, err error) error {
			if err == nil && info.Name() == "libslirp.so.0" {
				pathToSearch = path
			}
			return nil
		}); err != nil {
			log.Errorf("filepath.Walk: %v", err)
			return false
		}
		if pathToSearch == "" {
			log.Errorf("Not found libslirp.so.0")
			return false
		}
		elfFile, err := elf.Open(pathToSearch)
		if err != nil {
			log.Errorf("elf.Open: %v", err)
			return false
		}
		symbols, err := elfFile.DynamicSymbols()
		if err != nil {
			log.Errorf("elfFile.DynamicSymbols: %v", err)
			return false
		}
		for _, el := range symbols {
			if el.Name == "SLIRP_4.2" {
				log.Warn("SLIRP_4.2 and later do not allow to communicate between slirp interfaces")
				return false
			}
		}
		return true
	},
}

// RenderTemplate render Go template with Eden-related fuctions
func RenderTemplate(configFile string, tmpl string) (string, error) {
	var err error
	var buf bytes.Buffer

	viperLoaded, err := LoadConfigFile(configFile)
	if err != nil {
		log.Fatalf("error reading config: %s", err.Error())
		return "", err
	}
	if viperLoaded {
		t, err := template.New("Eden").Funcs(funcs).Parse(string(tmpl))
		if err != nil {
			return "", err
		}

		err = t.Execute(&buf, nil)
		if err != nil {
			return "", err
		}
		return buf.String(), err
	}
	return tmpl, err
}
