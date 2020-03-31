package main

import (
	"flag"
	"github.com/lf-edge/eden/pkg/utils"
	"log"
	"math/big"
	"net"
	"os"
	"path"
)

func main() {
	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	var dir string
	var dns string
	var ip string
	var uuid string
	flag.StringVar(&dir, "o", currentPath, "directory to save")
	flag.StringVar(&dns, "d", "mydomain.adam", "dns")
	flag.StringVar(&ip, "i", "192.168.0.1", "ip")
	flag.StringVar(&uuid, "u", "1", "ip")
	flag.Parse()
	rootCert, rootKey := utils.GenCARoot()
	ServerCert, ServerKey := utils.GenServerCert(rootCert, rootKey, big.NewInt(1), []net.IP{net.ParseIP(ip)}, []string{dns}, dns)
	ClientCert, ClientKey := utils.GenServerCert(rootCert, rootKey, big.NewInt(2), nil, nil, uuid)
	err = utils.WriteToFiles(rootCert, rootKey, path.Join(dir, "root-certificate.pem"), path.Join(dir, "root-certificate.key"))
	if err != nil {
		log.Fatal(err)
	}
	err = utils.WriteToFiles(ServerCert, ServerKey, path.Join(dir, "server.pem"), path.Join(dir, "server-key.pem"))
	if err != nil {
		log.Fatal(err)
	}
	err = utils.WriteToFiles(ClientCert, ClientKey, path.Join(dir, "onboard.cert.pem"), path.Join(dir, "onboard.key.pem"))
	if err != nil {
		log.Fatal(err)
	}
}
