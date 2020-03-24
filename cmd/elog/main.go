package main

import (
	"fmt"
	"github.com/itmo-eve/eden/pkg/elog"
	"io/ioutil"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s file [field:pattern ...]\n", os.Args[0])
		os.Exit(-1)
	}

	data, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		fmt.Println("File reading error", err)
		return
	}

	lb, err := elog.ParseLogBundle(data)
	if err != nil {
		fmt.Println("ParseLogBundle error", err)
		return
	}

	q := make(map[string]string)

	for _, a := range os.Args[2:] {
		s := strings.Split(a, ":")
		q[s[0]] = s[1]
	}

	for _, n := range lb.Log {
		//fmt.Println(n.Content)
		s := string(n.Content)
		le, err := elog.ParseLogItem(s)
		if err != nil {
			fmt.Println("ParseLogItem error", err)
			return
		}
		if elog.LogItemFind(le, q) == 1 {
			elog.LogPrn(&le)
		}
	}
}
