package main

import (
	"fmt"
	"os"
	"strings"
	"io/ioutil"
	"github.com/itmo-eve/eden/pkg/elog"
)

func main() {
	if (len(os.Args) < 2) {
		fmt.Printf("Usage: %s file [field:pattern ...]\n", os.Args[0])
		os.Exit(-1)
		}

	data, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		fmt.Println("File reading error", err)
		return
	}

	lb := elog.Parse_LogBundle (data)

	q := make(map[string]string)

	for _, a := range os.Args[2:] {
		s := strings.Split(a, ":")
		q[s[0]] = s[1]
	}

        for _, n := range lb.Log {
		//fmt.Println(n.Content)
		s := string(n.Content)
		le := elog.Parse_LogItem(s)
		if (elog.Find_LogItem(le, q) == 1) {
			elog.Log_prn(&le)
		}
        }
}
