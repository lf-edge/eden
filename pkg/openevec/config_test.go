package openevec_test

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"github.com/lf-edge/eden/pkg/openevec"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
)

type NestedConfig struct {
	NumField int `mapstructure:"numfield"`
}

type ServerConfig struct {
	Field   string            `mapstructure:"field"`
	Access  int               `mapstructure:"access"`
	HostFwd map[string]string `mapstructure:"hostfwd"`

	NestedField NestedConfig `mapstructure:"nested"`
}

type Config struct {
	Names     []string `mapstructure:"names"`
	IsSpecial bool     `mapstructure:"special"`

	Server ServerConfig `mapstructure:"server"`
}

func TestViperSerializeFromWriteConfig(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

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

			NestedField: NestedConfig{
				NumField: 21,
			},
		},
	}

	var buf bytes.Buffer
	openevec.WriteConfig(reflect.ValueOf(cfg), "", &buf, 0)

	v := viper.New()
	v.SetConfigType("yaml")
	err := v.ReadConfig(&buf)
	if err != nil {
		t.Errorf("error reading config: %v", err)
		return
	}

	// Unmarshal the configuration into the Config struct.
	gotCfg := &Config{}
	err = v.Unmarshal(&gotCfg)

	g.Expect(*gotCfg).To(BeEquivalentTo(cfg))
}

func TestConfigSliceType(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := Config{
		Names: []string{"test1", "test2"},
	}

	var buf bytes.Buffer
	openevec.WriteConfig(reflect.ValueOf(cfg), "", &buf, 0)

	v := viper.New()
	v.SetConfigType("yaml")
	err := v.ReadConfig(&buf)
	if err != nil {
		fmt.Println("error reading config:", err)
		return
	}

	gotCfg := &Config{}
	err = v.Unmarshal(&gotCfg)

	g.Expect(reflect.TypeOf(cfg.Names[0]).Kind()).To(BeEquivalentTo(reflect.String))
}
