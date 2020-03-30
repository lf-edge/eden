package main

import (
	"github.com/lf-edge/eden/pkg/controller"
	uuid "github.com/satori/go.uuid"
	"log"
)

func main() {
	cloudCxt := &controller.Ctx{}
	devID, _ := uuid.NewV4()
	err := cloudCxt.AddDevice(&devID)
	if err != nil {
		log.Fatal(err)
	}
	b, err := cloudCxt.GetConfigBytes(&devID)
	if err != nil {
		log.Fatal(err)
	}
	log.Print(string(b))
}
