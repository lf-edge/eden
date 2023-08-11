package templates

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/lf-edge/eden/pkg/utils"
	"github.com/spf13/viper"
)

func getConfig(t *testing.T) string {
	context, err := utils.ContextLoad()
	if err != nil {
		t.Fatalf("Load context error: %s", err)
	}
	configFile := context.GetCurrentConfig()
	if configFile == "" {
		configFile, err = utils.DefaultConfigPath()
		if err != nil {
			t.Fatalf("fail in DefaultConfigPath: %s", err)
		}
	}
	return configFile
}

func TestTemplateString(t *testing.T) {
	configFile := getConfig(t)
	viperLoaded, err := utils.LoadConfigFile(configFile)
	if err != nil {
		t.Fatalf("error reading config: %s", err.Error())
	}
	if viperLoaded {
		tests := map[string]string{
			"{{EdenConfig \"eden.root\"}}":                             viper.GetString("eden.root"),
			"{{EdenConfigPath \"eden.images.dist\"}}":                  utils.ResolveAbsPath(viper.GetString("eden.images.dist")),
			"{{$i := EdenConfig \"eden.images.dist\"}}{{EdenPath $i}}": utils.ResolveAbsPath(viper.GetString("eden.images.dist")),
		}

		for tmpl, res := range tests {
			out, err := utils.RenderTemplate(configFile, tmpl)
			if err != nil {
				t.Fatal(err)
			}
			if out != res {
				t.Fatalf("Template rendering error: '%s' != '%s'\n", out, res)
			}
		}
	}
}

func TestTemplateFile(t *testing.T) {
	configFile := getConfig(t)

	tmpl, err := os.ReadFile("template_test.tmpl")
	if err != nil {
		t.Fatal(err)
	}

	out, err := utils.RenderTemplate(configFile, string(tmpl))
	if err != nil {
		t.Fatal(err)
	}
	out = fmt.Sprintln(out)

	res, err := exec.Command("../../eden", "utils", "template", "template_test.tmpl").Output()
	if err != nil {
		t.Fatal(err)
	}

	if out != string(res) {
		t.Fatalf("Template rendering error. We got:\n'%s'\nMust be:\n'%s'\n", out, res)
	}
}
