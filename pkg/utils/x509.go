package utils

import (
	"crypto"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"io/ioutil"
	"math/big"

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
	return ParseFirstCertFromBlock(cert)
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

//ParseFirstCertFromBlock process provided certificate date
func ParseFirstCertFromBlock(b []byte) (*x509.Certificate, error) {
	certs, err := parseCertFromBlock(b)
	if err != nil {
		return nil, err
	}
	if len(certs) <= 0 {
		return nil, fmt.Errorf("no certs found")
	}

	return certs[0], nil
}

func parseCertFromBlock(b []byte) ([]*x509.Certificate, error) {
	var certs []*x509.Certificate
	for block, rest := pem.Decode(b); block != nil; block, rest = pem.Decode(rest) {
		if block.Type == "CERTIFICATE" {
			c, e := x509.ParseCertificates(block.Bytes)
			if e != nil {
				continue
			}
			certs = append(certs, c...)
		}
	}

	return certs, nil
}

func parsePrivateKey(keyPEMBlock []byte, passCode string) (interface{}, error) {
	var keyDERBlock *pem.Block
	var err error

	for len(keyPEMBlock) > 0 {
		keyDERBlock, keyPEMBlock = pem.Decode(keyPEMBlock)
		if keyDERBlock == nil {
			return nil, fmt.Errorf("No valid private key found")
		}
		pvtKeyBytes := keyDERBlock.Bytes

		if passCode != "" {
			pvtKeyBytes, err = x509.DecryptPEMBlock(keyDERBlock, []byte(passCode))
			if err != nil {
				return nil, fmt.Errorf("Error while decrypting the private key block: %v", err)
			}
		}
		switch keyDERBlock.Type {
		case "RSA PRIVATE KEY":
			privatekey, pErr := x509.ParsePKCS1PrivateKey(pvtKeyBytes) //PKCS1 standard
			if pErr != nil {
				return nil, fmt.Errorf("Unable to parse PKCS1 private key, %v", pErr)
			}
			return privatekey, nil
		case "PRIVATE KEY":
			//privatekeyPKCS8 can be RSA, ed25519, ecdsa.
			privatekeyPKCS8, pErr := x509.ParsePKCS8PrivateKey(pvtKeyBytes) //PKCS8 standard
			if pErr != nil {
				return nil, fmt.Errorf("Unable to parse PKCS8 private key, %v", pErr)
			}
			return privatekeyPKCS8, nil

		case "EC PRIVATE KEY":
			privateKey, err := x509.ParseECPrivateKey(pvtKeyBytes)
			if err != nil {
				return nil, fmt.Errorf("Unable to parse EC private key, %v", err)
			}
			return privateKey, nil
		}
	}
	return nil, fmt.Errorf("Unknown type of private key")
}

func ecdsakeyBytes(pubKey *ecdsa.PublicKey) (int, error) {
	curveBits := pubKey.Curve.Params().BitSize
	keyBytes := curveBits / 8
	if curveBits%8 > 0 {
		keyBytes++
	}

	if keyBytes%8 > 0 {
		return 0, fmt.Errorf("ecdsa pubkey size error, curveBits %v", curveBits)
	}
	return keyBytes, nil
}

// RSCombinedBytes - combine r & s into fixed length bytes
func rsCombinedBytes(rBytes, sBytes []byte, pubKey *ecdsa.PublicKey) ([]byte, error) {
	keySize, err := ecdsakeyBytes(pubKey)
	if err != nil {
		return nil, fmt.Errorf("RSCombinedBytes: ecdsa key bytes error %v", err)
	}
	rsize := len(rBytes)
	ssize := len(sBytes)
	if rsize > keySize || ssize > keySize {
		return nil, fmt.Errorf("RSCombinedBytes: error. keySize %v, rSize %v, sSize %v", keySize, rsize, ssize)
	}

	// basically the size is 32 bytes. the r and s needs to be both left padded to two 32 bytes slice
	// into a single signature buffer
	buffer := make([]byte, keySize*2)
	startPos := keySize - rsize
	copy(buffer[startPos:], rBytes)
	startPos = keySize*2 - ssize
	copy(buffer[startPos:], sBytes)
	return buffer[:], nil
}

func sha256FromECPoint(X, Y *big.Int, pubKey *ecdsa.PublicKey) ([32]byte, error) {
	var sha [32]byte
	bytes, err := rsCombinedBytes(X.Bytes(), Y.Bytes(), pubKey)
	if err != nil {
		return sha, fmt.Errorf("Error occurred while combining bytes for ECPoints: %v", err)
	}
	return sha256.Sum256(bytes), nil
}

func calculateSymmetricKeyForEcdhAES(deviceCert, controllerPrivateKey []byte) ([]byte, error) {
	//get public key of edge node.
	devCert, pErr := ParseFirstCertFromBlock(deviceCert)
	if pErr != nil {
		return nil, fmt.Errorf("Error in parsing public cert: %v", pErr)
	}

	var devPublicKey *ecdsa.PublicKey

	switch devCert.PublicKey.(type) {
	case *ecdsa.PublicKey:
		devPublicKey = devCert.PublicKey.(*ecdsa.PublicKey)
	default:
		return nil, fmt.Errorf("Public key type %v", "Unknown")
	}

	//get decoded private key.
	privateKey, rErr := parsePrivateKey(controllerPrivateKey, "")
	if rErr != nil {
		return nil, fmt.Errorf("Error in reading private key: %v", rErr)
	}

	var X, Y *big.Int
	switch privateKey := privateKey.(type) {
	case *ecdsa.PrivateKey:
		//multiply privateKey key with devPublic key.
		X, Y = elliptic.P256().Params().ScalarMult(devPublicKey.X, devPublicKey.Y, privateKey.D.Bytes())
	default:
		return nil, fmt.Errorf("unknown type of controller private key: %v", privateKey)
	}

	symmetricKey, err := sha256FromECPoint(X, Y, devPublicKey)
	if err != nil {
		return nil, err
	}
	return symmetricKey[:], nil
}

func computeSignatureWithCertAndKey(shaOfPayload, certPem, keyPem []byte) ([]byte, error) {
	var signature []byte
	var rsCombErr error

	cert, err := tls.X509KeyPair(certPem, keyPem)
	if err != nil {
		return nil, fmt.Errorf("computeSignatureWithCertAndKey X509KeyPair: %v", err)
	}
	switch key := cert.PrivateKey.(type) {

	case *ecdsa.PrivateKey:
		r, s, err := ecdsa.Sign(rand.Reader, key, shaOfPayload)
		if err != nil {
			return nil, err
		}
		signature, rsCombErr = rsCombinedBytes(r.Bytes(), s.Bytes(), &key.PublicKey)
		if rsCombErr != nil {
			return nil, rsCombErr
		}

	case *rsa.PrivateKey:
		var sErr error
		signature, sErr = rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, shaOfPayload)
		if sErr != nil {
			return nil, sErr
		}
	default:
		return nil, fmt.Errorf("signAuthData: privatekey default")

	}
	return signature, nil
}