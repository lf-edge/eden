package main

import (
	"github.com/lf-edge/eden/pkg/cloud"
	"github.com/lf-edge/eden/pkg/device"
	uuid "github.com/satori/go.uuid"
	"log"
)

func main() {
	cloudCxt := &cloud.Ctx{}
	devID, _ := uuid.NewV4()
	deviceCtx := device.CreateWithBaseConfig(devID, cloudCxt)
	b, err := deviceCtx.GenerateJSONBytes()
	if err != nil {
		log.Fatal(err)
	}
	log.Print(string(b))
}
