package utils

import (
	"errors"
	"log"
	"net"
	"strings"
)

func dockerSubnetPattern() (cmd string, args []string) {
	return "docker", strings.Split("network inspect bridge", " ")
}

func GetIPForDockerAccess() (ip string, err error) {
	dockerSubnetCmd, dockerSubnetArgs := dockerSubnetPattern()
	cmdOut, cmdErr, err := RunCommandAndWait(dockerSubnetCmd, dockerSubnetArgs...)
	if err != nil {
		log.Print(cmdOut)
		log.Print(cmdErr)
		log.Print("Probably you have no access do docker socket or no configured network")
		return "", err
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Fatal(err)
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				if strings.Contains(cmdOut, ipnet.IP.String()) {
					continue
				}
				ip = ipnet.IP.String()
			}
		}
	}
	if ip == "" {
		return "", errors.New("no IP found")
	}
	return ip, nil
}
