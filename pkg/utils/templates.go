package utils

import (
	"bytes"
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
	"EdenConfigPath":  func(path string) string {
		res := ResolveAbsPath(viper.GetString(path))
		return res
	},
	// Resolve path relative to the Eden root directory  
	"EdenPath":  func(path string) string {
		res := ResolveAbsPath(path)
		return res
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
	} else {
		return tmpl, err
	}
}
