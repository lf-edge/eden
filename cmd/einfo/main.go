package main

import (
	"os"
	"io/ioutil"
	"fmt"
	"github.com/itmo-eve/eden/pkg/einfo"
)


func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s file\n", os.Args[0])
		os.Exit(-1)
	}

	data, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		fmt.Println("File reading error", err)
		return
	}

	im, err := einfo.ParseZInfoMsg(data)
	if err != nil {
		fmt.Println("ParseLogBundle error", err)
		return
	}

	fmt.Printf("%q", im)
	einfo.InfoPrn(&im)
}
