package utils

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"net"
	"os"
	"time"

	"github.com/lf-edge/eden/pkg/defaults"
)

func genCertECDSA(template, parent *x509.Certificate, publicKey *ecdsa.PublicKey, privateKey *rsa.PrivateKey) *x509.Certificate {
	certBytes, err := x509.CreateCertificate(rand.Reader, template, parent, publicKey, privateKey)
	if err != nil {
		panic("Failed to create certificate:" + err.Error())
	}

	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		panic("Failed to parse certificate:" + err.Error())
	}

	return cert
}

func genCert(template, parent *x509.Certificate, publicKey *rsa.PublicKey, privateKey *rsa.PrivateKey) *x509.Certificate {
	certBytes, err := x509.CreateCertificate(rand.Reader, template, parent, publicKey, privateKey)
	if err != nil {
		panic("Failed to create certificate:" + err.Error())
	}

	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		panic("Failed to parse certificate:" + err.Error())
	}

	return cert
}

//GenCARoot gen root CA
func GenCARoot() (*x509.Certificate, *rsa.PrivateKey) {
	var rootTemplate = x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Country:      []string{defaults.DefaultX509Country},
			Organization: []string{defaults.DefaultX509Company},
			CommonName:   "Root CA",
		},
		NotBefore:             time.Now().Add(-10 * time.Second),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            2,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}
	priv, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		panic(err)
	}
	rootCert := genCert(&rootTemplate, &rootTemplate, &priv.PublicKey, priv)
	return rootCert, priv
}

//GenServerCert cert gen
func GenServerCert(cert *x509.Certificate, key *rsa.PrivateKey, serial *big.Int, ip []net.IP, dns []string, CN string) (*x509.Certificate, *rsa.PrivateKey) {
	priv, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		panic(err)
	}

	var ServerTemplate = x509.Certificate{
		SerialNumber:   serial,
		NotBefore:      time.Now().Add(-10 * time.Second),
		NotAfter:       time.Now().AddDate(10, 0, 0),
		KeyUsage:       x509.KeyUsageCRLSign,
		ExtKeyUsage:    []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IsCA:           false,
		MaxPathLenZero: true,
		IPAddresses:    ip,
		DNSNames:       dns,
		Subject: pkix.Name{
			Country:      []string{defaults.DefaultX509Country},
			Organization: []string{defaults.DefaultX509Company},
			CommonName:   CN,
		},
	}

	ServerCert := genCert(&ServerTemplate, cert, &priv.PublicKey, key)
	return ServerCert, priv

}

//GenServerCertElliptic elliptic cert
func GenServerCertElliptic(cert *x509.Certificate, key *rsa.PrivateKey, serial *big.Int, ip []net.IP, dns []string, uuid string) (*x509.Certificate, *ecdsa.PrivateKey) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}

	var ServerTemplate = x509.Certificate{
		SerialNumber:   serial,
		NotBefore:      time.Now().Add(-10 * time.Second),
		NotAfter:       time.Now().AddDate(10, 0, 0),
		KeyUsage:       x509.KeyUsageCRLSign,
		ExtKeyUsage:    []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IsCA:           false,
		MaxPathLenZero: true,
		IPAddresses:    ip,
		DNSNames:       dns,
		Subject: pkix.Name{
			Country:      []string{defaults.DefaultX509Country},
			Organization: []string{defaults.DefaultX509Company},
			CommonName:   uuid,
		},
	}

	ServerCert := genCertECDSA(&ServerTemplate, cert, &priv.PublicKey, key)
	return ServerCert, priv

}

//WriteToFiles write cert and key
func WriteToFiles(crt *x509.Certificate, key interface{}, certFile string, keyFile string) (err error) {
	certOut, err := os.Create(certFile)
	if err != nil {
		return err
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: crt.Raw}); err != nil {
		return err
	}
	if err := certOut.Close(); err != nil {
		return err
	}

	keyOut, err := os.OpenFile(keyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	var privBytes []byte
	switch keyTyped := key.(type) {
	case *rsa.PrivateKey:
		privBytes, err = x509.MarshalPKCS8PrivateKey(keyTyped)
		if err != nil {
			return err
		}
		if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
			return err
		}
	case *ecdsa.PrivateKey:
		privBytes, err = x509.MarshalECPrivateKey(keyTyped)
		if err != nil {
			return err
		}
		secp256r1, err := asn1.Marshal(asn1.ObjectIdentifier{1, 2, 840, 10045, 3, 1, 7})
		if err != nil {
			return err
		}
		if err := pem.Encode(keyOut, &pem.Block{Type: "EC PARAMETERS", Bytes: secp256r1}); err != nil {
			return err
		}
		if err := pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes}); err != nil {
			return err
		}
	default:
		return errors.New("unknown key format")
	}
	return keyOut.Close()
}

// ParseCertificate from file
func ParseCertificate(certFile string) (*x509.Certificate, error) {
	cert, err := ioutil.ReadFile(certFile)
	if err != nil {
		return nil, fmt.Errorf("cannot read file with certificate: %s", err)
	}
	pemBlock, _ := pem.Decode(cert)
	return x509.ParseCertificate(pemBlock.Bytes)
}

// ParsePrivateKey from file
func ParsePrivateKey(keyFile string) (*rsa.PrivateKey, error) {
	key, err := ioutil.ReadFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("cannot read file with private key: %s", err)
	}
	pemBlock, _ := pem.Decode(key)
	var parsedKey interface{}
	if parsedKey, err = x509.ParsePKCS8PrivateKey(pemBlock.Bytes); err != nil {
		return nil, fmt.Errorf("cannot parse private key: %s", err)
	}
	privateKey, ok := parsedKey.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("cannot parse private key: wrong type")
	}
	return privateKey, nil
}
