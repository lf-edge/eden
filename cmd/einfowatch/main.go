package main

import (
	"fmt"
	"os"
	"github.com/itmo-eve/eden/pkg/einfo"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s dir\n", os.Args[0])
		os.Exit(-1)
	}

	einfo.InfoWatch(os.Args[1], einfo.HandleAll)
}
