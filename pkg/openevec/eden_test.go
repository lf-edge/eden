package openevec_test

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/lf-edge/eden/pkg/openevec"
	"gotest.tools/assert"
)

func TestParseTemplateFile(t *testing.T) {
	var buf bytes.Buffer

	f, err := os.CreateTemp("", "template.*.tmpl")
	if err != nil {
		t.Errorf("CreateTemp failed %v", err)
		return
	}
	defer f.Close()

	const tmpl = "{{.Eden.Root}} {{.Eden.BinDir}}"
	f.Write([]byte(tmpl))

	const rootVal = "rv"
	const binDirVal = "bdv"
	cfg := openevec.EdenSetupArgs{
		Eden: openevec.EdenConfig{
			Root:   rootVal,
			BinDir: binDirVal,
		},
	}

	if err = openevec.ParseTemplateFile(f.Name(), cfg, &buf); err != nil {
		t.Errorf("parseTemplateFile failed: %v", err)
		return
	}

	assert.Equal(t, buf.String(), fmt.Sprintf("%s %s", rootVal, binDirVal))
}
