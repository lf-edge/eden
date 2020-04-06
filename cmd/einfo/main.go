package main

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/controller/einfo"
	"io/ioutil"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s file [field:regexp ...]\n", os.Args[0])
		os.Exit(-1)
	}

	data, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		fmt.Println("File reading error", err)
		return
	}

	q := make(map[string]string)
	for _, a := range os.Args[2:] {
		s := strings.Split(a, ":")
		q[s[0]] = s[1]
	}

	im, err := einfo.ParseZInfoMsg(data)
	if err != nil {
		fmt.Println("ParseZInfoMsg error", err)
		return
	}

	ds := einfo.ZInfoFind(&im, q, einfo.ZInfoDevSW)
	if ds != nil {
		einfo.ZInfoPrn(&im, ds, einfo.ZInfoDevSW)
	}
}
