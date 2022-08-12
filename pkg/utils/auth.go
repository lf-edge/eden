package utils

import (
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/lf-edge/eve/api/go/auth"
	"github.com/lf-edge/eve/api/go/certs"
	"github.com/lf-edge/eve/api/go/evecommon"
)

func PrepareAuthContainer(
	payload []byte, signingCertPath, signingKeyPath string) (*auth.AuthContainer, error) {
	authContainer := &auth.AuthContainer{}

	//get sender cert detail
	var senderCertHash []byte
	var signingCert []byte

	certChain, gErr := LoadCertChain(signingCertPath, certs.ZCertType_CERT_TYPE_CONTROLLER_SIGNING)
	if gErr != nil {
		return nil, gErr
	}
	for _, cert := range certChain {
		if cert.Type == certs.ZCertType_CERT_TYPE_CONTROLLER_SIGNING {
			senderCertHash = cert.CertHash
			signingCert = cert.Cert
		}
	}

	//read private signing key.
	signingPrivateKey, rErr := ioutil.ReadFile(signingKeyPath)
	if rErr != nil {
		return nil, fmt.Errorf("error occurred while reading signing key: %v", rErr)
	}

	//compute hash of payload
	hashedPayload := sha256.Sum256(payload)

	//compute signature of payload hash
	signatureOfPayloadHash, scErr := computeSignatureWithCertAndKey(
		hashedPayload[:], signingCert, signingPrivateKey)
	if scErr != nil {
		return nil, fmt.Errorf("error occurred while computing signature: %v", scErr)
	}

	authBody := new(auth.AuthBody)
	authBody.Payload = payload
	authContainer.ProtectedPayload = authBody
	authContainer.Algo = evecommon.HashAlgorithm_HASH_ALGORITHM_SHA256_32BYTES
	authContainer.SenderCertHash = senderCertHash
	authContainer.SignatureHash = signatureOfPayloadHash
	return authContainer, nil
}

func LoadCertChain(certPath string, certType certs.ZCertType) ([]*certs.ZCert, error) {
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		return nil, err
	}
	certData, err := ioutil.ReadFile(certPath)
	if err != nil {
		return nil, err
	}

	var certChain []*certs.ZCert
	//split certificates from file.
	certsArr := strings.SplitAfter(string(certData), "-----END CERTIFICATE-----")
	for _, cert := range certsArr {
		certsAfterTrim := strings.TrimSpace(cert)
		if len(certsAfterTrim) == 0 {
			continue
		}
		individualCert := []byte(certsAfterTrim)
		shaOfCert := sha256.Sum256(individualCert)

		certDetail := &certs.ZCert{}
		certDetail.Cert = individualCert
		certDetail.CertHash = shaOfCert[:]

		parsedCert, pErr := parseCertFromBlock(individualCert)
		if pErr != nil {
			return nil, pErr
		}
		for _, pVal := range parsedCert {
			if !pVal.IsCA {
				certDetail.Type = certType
			} else {
				certDetail.Type = certs.ZCertType_CERT_TYPE_CONTROLLER_INTERMEDIATE
			}
		}
		certChain = append(certChain, certDetail)
	}
	return certChain, nil
}