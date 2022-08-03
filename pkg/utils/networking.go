package utils

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/lf-edge/eden/pkg/defaults"
	log "github.com/sirupsen/logrus"
)

//IFInfo stores information about net address and subnet
type IFInfo struct {
	Subnet        *net.IPNet
	FirstAddress  net.IP
	SecondAddress net.IP
}

func getSubnetByInd(ind int) (*net.IPNet, error) {
	if ind < 0 || ind > 255 {
		return nil, fmt.Errorf("error in index %d", ind)
	}
	_, curNet, err := net.ParseCIDR(fmt.Sprintf("192.168.%d.1/24", ind))
	return curNet, err
}

func getIPByInd(ind int) ([]net.IP, error) {
	if ind < 0 || ind > 255 {
		return nil, fmt.Errorf("error in index %d", ind)
	}
	IP := net.ParseIP(fmt.Sprintf("192.168.%d.10", ind))
	if IP == nil {
		return nil, fmt.Errorf("error in ParseIP for index %d", ind)
	}
	ips := []net.IP{IP}
	IP2 := net.ParseIP(fmt.Sprintf("192.168.%d.11", ind))
	if IP2 == nil {
		return nil, fmt.Errorf("error in ParseIP for index %d", ind)
	}
	ips = append(ips, IP2)
	return ips, nil
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
			ips, err := getIPByInd(curSubnetInd)
			if err != nil {
				return nil, fmt.Errorf("error in getIPByInd: %s", err)
			}
			result = append(result, IFInfo{
				Subnet:        curNet,
				FirstAddress:  ips[0],
				SecondAddress: ips[1],
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
		log.Errorf("GetDockerNetworks: %s", err)
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

//GetFileSizeURL returns file size for url
func GetFileSizeURL(url string) int64 {
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

//RepeatableAttempt do request several times waiting for nil error and expected status code
func RepeatableAttempt(client *http.Client, req *http.Request) (response *http.Response, err error) {
	maxRepeat := defaults.DefaultRepeatCount
	delayTime := defaults.DefaultRepeatTimeout

	for i := 0; i < maxRepeat; i++ {
		timer := time.AfterFunc(2*delayTime, func() {
			i = 0
		})
		resp, err := client.Do(req)
		wrongCode := false
		if err == nil {
			// we should check the status code of the response and try again if needed
			if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusAccepted {
				return resp, nil
			}
			wrongCode = true
			buf, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Debugf("bad status: %s", resp.Status)
			} else {
				log.Debugf("bad status (%s) in response (%s)", resp.Status, string(buf))
			}
		}
		log.Debugf("error %s URL %s: %v", req.Method, req.RequestURI, err)
		timer.Stop()
		if wrongCode {
			log.Infof("Received unexpected StatusCode(%s): repeat request (%d) of (%d)",
				http.StatusText(resp.StatusCode), i, maxRepeat)
		} else {
			log.Infof("Attempt to re-establish connection (%d) of (%d)", i, maxRepeat)
		}
		time.Sleep(delayTime)
	}
	return nil, fmt.Errorf("all connection attempts failed")
}

//UploadFile send file in form
func UploadFile(client *http.Client, url, filePath, prefix string) (result *http.Response, err error) {
	body, writer := io.Pipe()

	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}

	mwriter := multipart.NewWriter(writer)
	req.Header.Add("Content-Type", mwriter.FormDataContentType())

	errchan := make(chan error)

	fileName := filepath.Base(filePath)
	if prefix != "" {
		fileName = fmt.Sprintf("%s/%s", prefix, fileName)
	}

	go func() {
		defer writer.Close()
		defer mwriter.Close()
		w, err := mwriter.CreateFormFile("file", fileName)
		if err != nil {
			errchan <- err
			return
		}
		in, err := os.Open(filePath)
		if err != nil {
			errchan <- err
			return
		}
		defer in.Close()

		counter := &writeCounter{step: 10 * 1024 * 1024, message: "Uploading..."}
		if written, err := io.Copy(w, io.TeeReader(in, counter)); err != nil {
			errchan <- fmt.Errorf("error copying %s (%d bytes written): %v", filePath, written, err)
			return
		}
		fmt.Printf("\n")

		if err := mwriter.Close(); err != nil {
			errchan <- err
			return
		}
		log.Info("Waiting for SHA256 calculation")
	}()
	respchan := make(chan *http.Response)
	go func() {
		resp, err := client.Do(req)
		if err != nil {
			errchan <- err
		} else {
			respchan <- resp
		}
	}()
	var merr error
	var resp *http.Response
	select {
	case merr = <-errchan:
		return nil, fmt.Errorf("http/multipart error: %v", merr)
	case resp = <-respchan:
		return resp, nil
	}
}

// FindUnusedPort : find port number not currently used by the host.
func FindUnusedPort() (uint16, error) {
	// We let the kernel to find the port for us.
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return uint16(l.Addr().(*net.TCPAddr).Port), nil
}
