// Command nim-bootstrap-pb-gen generates a signed BootstrapConfig
// protobuf from a JSON-encoded EdgeDevConfig.
//
// Usage:
//
//	nim-bootstrap-pb-gen <input-json> <output-pb>
//
// The output is suitable for placing at /config/bootstrap-config.pb on an
// EVE device. It is signed with the controller signing key under
// $EDEN_HOME/certs/ — the same key that eden setup would use. Tests
// under tests/network/testdata/nim_bootstrap_*.txt invoke this helper to
// inject controllable bootstrap configs at runtime so they can verify
// content-level round-trip behavior (e.g. that a port with a distinctive
// Logicallabel from the bootstrap pb appears in NIM's
// DevicePortConfigList).
//
// This is a standalone repackaging of the bootstrap-pb generation logic
// in pkg/eden/eden.go's GenerateEVEConfig (around lines 683–725 as of
// the time of writing). Keeping it in a separate binary avoids the need
// to re-run eden setup for every test.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve-api/go/certs"
	"github.com/lf-edge/eve-api/go/config"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr,
			"usage: %s <input-json> <output-pb>\n"+
				"  reads a JSON-encoded EdgeDevConfig from <input-json>,\n"+
				"  signs it with eden's controller signing key (from\n"+
				"  $EDEN_HOME/certs/), and writes a serialized BootstrapConfig\n"+
				"  protobuf to <output-pb>.\n",
			os.Args[0])
		os.Exit(2)
	}
	in, out := os.Args[1], os.Args[2]

	payload, err := os.ReadFile(in)
	if err != nil {
		log.Fatalf("read %s: %v", in, err)
	}
	var devConf config.EdgeDevConfig
	if err := protojson.Unmarshal(payload, &devConf); err != nil {
		log.Fatalf("parse %s as JSON EdgeDevConfig: %v", in, err)
	}
	// Stamp a fresh timestamp so the bootstrap-pb is "newer" than any
	// pre-existing controller config the device may have cached, mirroring
	// what eden setup --eve-bootstrap-file does.
	devConf.ConfigTimestamp = timestamppb.New(time.Now())

	devConfPbuf, err := proto.Marshal(&devConf)
	if err != nil {
		log.Fatalf("marshal EdgeDevConfig: %v", err)
	}

	edenHome, err := utils.DefaultEdenDir()
	if err != nil {
		log.Fatalf("default eden dir: %v", err)
	}
	globalCertsDir := filepath.Join(edenHome, defaults.DefaultCertsDist)
	signingCertPath := filepath.Join(globalCertsDir, "signing.pem")
	signingKeyPath := filepath.Join(globalCertsDir, "signing-key.pem")

	signed, err := utils.PrepareAuthContainer(devConfPbuf, signingCertPath, signingKeyPath)
	if err != nil {
		log.Fatalf("PrepareAuthContainer: %v", err)
	}
	controllerCerts, err := utils.LoadCertChain(
		signingCertPath, certs.ZCertType_CERT_TYPE_CONTROLLER_SIGNING)
	if err != nil {
		log.Fatalf("LoadCertChain: %v", err)
	}

	bootstrap := &config.BootstrapConfig{
		SignedConfig:    signed,
		ControllerCerts: controllerCerts,
	}
	pb, err := proto.Marshal(bootstrap)
	if err != nil {
		log.Fatalf("marshal BootstrapConfig: %v", err)
	}
	if err := os.WriteFile(out, pb, 0644); err != nil {
		log.Fatalf("write %s: %v", out, err)
	}
}
