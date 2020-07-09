package utils

import (
	"fmt"
	"github.com/amitbet/vncproxy/client"
	"github.com/amitbet/vncproxy/logger"
	"net"
)

//GetDesktopName return DesktopName from VNC server address with password (if not empty)
func GetDesktopName(address string, password string) (string, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return "", fmt.Errorf("fail in connect to VNC: %s", err)
	}
	logger.SetLogLevel("fatal")
	var noAuth client.ClientAuthNone
	var authArr []client.ClientAuth
	if password == "" {
		authArr = append(authArr, &noAuth)
	} else {
		authArr = append(authArr, &client.PasswordAuth{Password: password})
	}
	configVNCClient := &client.ClientConfig{
		Auth:      authArr,
		Exclusive: false,
	}
	connVNC, err := client.NewClientConn(conn, configVNCClient)
	if err != nil {
		return "", fmt.Errorf("fail in NewClientConn: %s", err)
	}
	if err = connVNC.Connect(); err != nil {
		return "", fmt.Errorf("fail in connect with NewClientConn: %s", err)
	}
	defer connVNC.Close()
	desktopName := connVNC.DesktopName
	return desktopName, nil
}
