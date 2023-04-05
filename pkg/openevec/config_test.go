package openevec_test

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"github.com/lf-edge/eden/pkg/openevec"
	"github.com/spf13/viper"
)

type ServerConfig struct {
	Field   string            `mapstructure:"field"`
	Access  int               `mapstructure:"access"`
	HostFwd map[string]string `mapstructure:"hostfwd"`
}

type Config struct {
	Names     []string `mapstructure:"names"`
	IsSpecial bool     `mapstructure:"special"`

	Server ServerConfig `mapstructure:"server"`
}

func (lhs *Config) IsEqual(rhs Config) bool {
	for _, lname := range lhs.Names {
		contains := false
		for _, rname := range rhs.Names {
			if lname == rname {
				contains = true
				break
			}
		}
		if !contains {
			//fmt.Println("Missing ", lname)
			return false
		}
	}

	if lhs.IsSpecial != rhs.IsSpecial {
		//fmt.Println("IsSpecial missmatch")
		return false
	}

	if lhs.Server.Field != rhs.Server.Field {
		//fmt.Println("Server Field missmatch")
		return false
	}

	if lhs.Server.Access != rhs.Server.Access {
		//fmt.Println("Server Access missmatch")
		return false
	}

	for k, lval := range lhs.Server.HostFwd {
		if rval, ok := rhs.Server.HostFwd[k]; ok {
			if lval != rval {
				//fmt.Println("Key %v missmatch. Have %v got %v", k, lhs, rhs)
				return false
			}
		} else {
			//fmt.Println("Missing hostfwd key ", k)
			return false
		}
	}

	return true
}

func TestViperSerializeFromWriteConfig(t *testing.T) {
	cfg := Config{
		Names:     []string{"test1", "test2"},
		IsSpecial: false,

		Server: ServerConfig{
			Field:  "ServerField",
			Access: 42,

			HostFwd: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
	}

	var buf bytes.Buffer
	openevec.WriteConfig(reflect.ValueOf(cfg), &buf, 0)

	v := viper.New()
	v.SetConfigType("yaml")
	err := v.ReadConfig(&buf)
	if err != nil {
		fmt.Println("error reading config:", err)
		return
	}

	// Unmarshal the configuration into the Config struct.
	gotCfg := &Config{}
	err = v.Unmarshal(&gotCfg)
	if !gotCfg.IsEqual(cfg) {
		t.Errorf("Generated config is = %v; want %v", gotCfg, cfg)
	}
}
