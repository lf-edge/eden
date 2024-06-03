package openevec_test

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/lf-edge/eden/pkg/openevec"
	"github.com/onsi/gomega"
)

func TestParseTemplateFile(t *testing.T) {
	t.Parallel()

	g := gomega.NewGomegaWithT(t)

	var buf bytes.Buffer

	f, err := os.CreateTemp("", "template.*.tmpl")
	if err != nil {
		t.Errorf("CreateTemp failed %v", err)
		return
	}
	defer f.Close()

	const tmpl = "{{.Eden.Root}} {{.Eden.BinDir}}"
	_, err = f.Write([]byte(tmpl))
	g.Expect(err).To(gomega.BeNil())

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

	g.Expect(buf.String()).To(gomega.BeEquivalentTo(fmt.Sprintf("%s %s", rootVal, binDirVal)))
}
