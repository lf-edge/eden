package server

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func (s *EServer) serveSFTP(listener net.Listener, errorChan chan error) {
	config := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if c.User() == s.User && string(pass) == s.Password {
				log.Printf("serveSFTP: login: %s\n", c.User())
				return nil, nil
			}
			return nil, fmt.Errorf("serveSFTP: password rejected for %q", c.User())
		},
	}

	serverOptions := []sftp.ServerOption{
		sftp.WithDebug(os.Stderr),
	}

	if s.ReadOnly {
		serverOptions = append(serverOptions, sftp.ReadOnly())
	}

	privateBytes, err := os.ReadFile("/root/.ssh/id_rsa")
	if err != nil {
		errorChan <- fmt.Errorf("serveSFTP: failed load private key: %s", err)
		return
	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		errorChan <- fmt.Errorf("serveSFTP: failed to parse private key: %s", err)
		return
	}

	config.AddHostKey(private)

	for {
		nConn, err := listener.Accept()
		if err != nil {
			log.Printf("serveSFTP: failed to accept incoming connection: %s\n", err)
			continue
		}
		go func(conn net.Conn) {
			_, channel, reqs, err := ssh.NewServerConn(nConn, config)
			if err != nil {
				log.Printf("serveSFTP: failed to handshake: %s\n", err)
				return
			}

			// The incoming Request channel must be serviced.
			go ssh.DiscardRequests(reqs)

			// Service the incoming Channel channel.
			for newChannel := range channel {
				if newChannel.ChannelType() != "session" {
					if err := newChannel.Reject(ssh.UnknownChannelType, "unknown channel type"); err != nil {
						log.Println("serveSFTP: could not reject channel.", err)
						return
					}
					log.Printf("serveSFTP: unknown channel type: %s\n", newChannel.ChannelType())
					continue
				}
				channel, requests, err := newChannel.Accept()
				if err != nil {
					log.Println("serveSFTP: could not accept channel.", err)
					return
				}

				go func(in <-chan *ssh.Request) {
					for req := range in {
						ok := false
						switch req.Type {
						case "subsystem":
							if string(req.Payload[4:]) == "sftp" {
								ok = true
							}
						}
						if err := req.Reply(ok, nil); err != nil {
							log.Println("serveSFTP: cannot reply:", err)
							return
						}
					}
				}(requests)

				server, err := sftp.NewServer(
					channel,
					serverOptions...,
				)
				if err != nil {
					log.Printf("serveSFTP: NewServer error: %s\n", err)
					return
				}
				if err := server.Serve(); err == io.EOF {
					if err := server.Close(); err != nil {
						log.Printf("serveSFTP: cannot close server: %s\n", err)
					}
					log.Println("serveSFTP: sftp client exited session.")
				} else if err != nil {
					log.Printf("serveSFTP: sftp server completed with error: %s\n", err)
				}
			}
		}(nConn)
	}
}
