package utils

import (
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func dockerSubnetPattern() (cmd string, args []string) {
	return "docker", strings.Split("network inspect bridge", " ")
}

//IFInfo stores information about net address and subnet
type IFInfo struct {
	Subnet       *net.IPNet
	FirstAddress net.IP
}

func getSubnetByInd(ind int) (*net.IPNet, error) {
	if ind < 0 || ind > 255 {
		return nil, fmt.Errorf("error in index %d", ind)
	}
	_, curNet, err := net.ParseCIDR(fmt.Sprintf("192.168.%d.1/24", ind))
	return curNet, err
}

func getIPByInd(ind int) (net.IP, error) {
	if ind < 0 || ind > 255 {
		return nil, fmt.Errorf("error in index %d", ind)
	}
	IP := net.ParseIP(fmt.Sprintf("192.168.%d.10", ind))
	if IP == nil {
		return nil, fmt.Errorf("error in ParseIP for index %d", ind)
	}
	return IP, nil
}

//GetSubnetsNotUsed prepare map with subnets and ip not used by any interface of host
func GetSubnetsNotUsed(count int) ([]IFInfo, error) {
	var result []IFInfo
	curSubnetInd := 0
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}
	for ; len(result) < count; curSubnetInd++ {
		curNet, err := getSubnetByInd(curSubnetInd)
		if err != nil {
			return nil, fmt.Errorf("error in GetSubnetsNotUsed: %s", err)
		}
		contains := false
		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					if curNet.Contains(ipnet.IP) {
						contains = true
						break
					}
				}
			}
		}
		if !contains {
			ip, err := getIPByInd(curSubnetInd)
			if err != nil {
				return nil, fmt.Errorf("error in getIPByInd: %s", err)
			}
			result = append(result, IFInfo{
				Subnet:       curNet,
				FirstAddress: ip,
			})
		}
	}
	return result, nil
}

//GetIPForDockerAccess is service function to obtain IP for adam access
//The function is filter out docker bridge
func GetIPForDockerAccess() (ip string, err error) {
	networks, err := GetDockerNetworks()
	if err != nil {
		return "", fmt.Errorf("GetDockerNetworks: %s", err)
	}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Fatal(err)
	}
out:
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				for _, el := range networks {
					if el.Contains(ipnet.IP) {
						continue out
					}
				}
				ip = ipnet.IP.String()
				break
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

//GetSubnetIPs return all IPs from subnet
func GetSubnetIPs(subnet string) (result []net.IP) {
	ip, ipnet, err := net.ParseCIDR(subnet)
	if err != nil {
		log.Fatal(err)
	}
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		result = append(result, net.ParseIP(ip.String()))
	}
	return
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

//GetFileSizeUrl returns file size for url
func GetFileSizeUrl(url string) int64 {
	resp, err := http.Head(url)
	if err != nil {
		log.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		log.Fatal(resp.Status)
	}
	size, _ := strconv.Atoi(resp.Header.Get("Content-Length"))
	return int64(size)
}
