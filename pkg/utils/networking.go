package utils

import (
	"errors"
	"log"
	"net"
	"net/url"
	"strings"
)

func dockerSubnetPattern() (cmd string, args []string) {
	return "docker", strings.Split("network inspect bridge", " ")
}

//GetIPForDockerAccess is service function to obtain IP for adam access
//The function is filter out docker bridge
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

//ResolveURL concatenate parts of url
func ResolveURL(b, p string) (string, error) {
	u, err := url.Parse(p)
	if err != nil {
		return "", err
	}
	base, err := url.Parse(b)
	if err != nil {
		return "", err
	}
	return base.ResolveReference(u).String(), nil
}
