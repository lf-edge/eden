package main

import (
	"fmt"
	"github.com/itmo-eve/eden/pkg/einfo"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s dir [field:pattern ...]\n", os.Args[0])
		os.Exit(-1)
	}

	q := make(map[string]string)

	for _, a := range os.Args[2:] {
		s := strings.Split(a, ":")
		q[s[0]] = s[1]
	}

	einfo.InfoWatch(os.Args[1], q, einfo.ZInfoDevSWFind, einfo.HandleAll)
}
