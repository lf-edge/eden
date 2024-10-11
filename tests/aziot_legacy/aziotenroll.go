package aziot

import (
	"bufio"
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/go-tpm/tpmutil"
	etpm "github.com/lf-edge/eve/pkg/pillar/evetpm"
)

// Create, Update, Delete Enrollment API URL
const enrollmentAPIURL = "https://%s/enrollments/%s?api-version=2021-10-01"

// getProvisioningService gets the provisioning service name from the IoT DPS connection string.
func getProvisioningService(connStr string) (hostName string, err error) {
	parts := strings.Split(connStr, ";")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid connection string")
	}

	for _, part := range parts {
		if strings.HasPrefix(part, "HostName=") {
			hostName = strings.TrimPrefix(part, "HostName=")
			break
		}
	}

	if hostName == "" {
		return "", fmt.Errorf("invalid connection string")
	}

	return hostName, nil
}

// parseConnectionString parses the IoT DPS connection string and returns its components.
func parseConnectionString(connStr string) (hostName, sharedAccessKeyName, sharedAccessKey string, err error) {
	parts := strings.Split(connStr, ";")
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("invalid connection string")
	}

	for _, part := range parts {
		if strings.HasPrefix(part, "HostName=") {
			hostName = strings.TrimPrefix(part, "HostName=")
		} else if strings.HasPrefix(part, "SharedAccessKeyName=") {
			sharedAccessKeyName = strings.TrimPrefix(part, "SharedAccessKeyName=")
		} else if strings.HasPrefix(part, "SharedAccessKey=") {
			sharedAccessKey = strings.TrimPrefix(part, "SharedAccessKey=")
		}
	}

	if hostName == "" || sharedAccessKeyName == "" || sharedAccessKey == "" {
		return "", "", "", fmt.Errorf("invalid connection string")
	}

	return hostName, sharedAccessKeyName, sharedAccessKey, nil
}

// generateSasToken generates a SAS token for Azure IoT DPS
func generateSasToken(uri, keyName, key string, expiry int64) (string, error) {
	encodedURI := url.QueryEscape(uri)
	stringToSign := fmt.Sprintf("%s\n%d", encodedURI, expiry)
	keyBytes, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return "", err
	}

	h := hmac.New(sha256.New, keyBytes)
	h.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	token := fmt.Sprintf("SharedAccessSignature sr=%s&sig=%s&se=%d&skn=%s", encodedURI, url.QueryEscape(signature), expiry, keyName)
	return token, nil
}

func getSasTokenFromConnectionString(connectionString string, hours uint) (string, error) {
	// Parse the connection string
	hostName, sharedAccessKeyName, sharedAccessKey, err := parseConnectionString(connectionString)
	if err != nil {
		return "", fmt.Errorf("failed to parse connection string: %v", err)
	}

	expiry := (time.Now().Unix() + 3600) * int64(hours)
	sasToken, err := generateSasToken(hostName, sharedAccessKeyName, sharedAccessKey, expiry)
	if err != nil {
		return "", fmt.Errorf("failed to generate SAS token: %v", err)
	}

	return sasToken, nil
}

// addTPMEnrollment adds a new TPM enrollment to Azure IoT DPS.
func addTPMEnrollment(enrollmentID, endorsementKey, provService, sasToken string) error {
	url := fmt.Sprintf(enrollmentAPIURL, provService, enrollmentID)

	// Prepare the enrollment body for TPM attestation
	enrollment := map[string]interface{}{
		"registrationId": enrollmentID,
		"attestation": map[string]interface{}{
			"type": "tpm",
			"tpm": map[string]interface{}{
				"endorsementKey": endorsementKey,
			},
		},
		"provisioningStatus": "enabled",
		"allocationPolicy":   "hashed",
		"capabilities": map[string]bool{
			"iotEdge": false,
		},
		"reprovisionPolicy": map[string]bool{
			"migrateDeviceData":   true,
			"updateHubAssignment": true,
		},
	}

	enrollmentJSON, err := json.Marshal(enrollment)
	if err != nil {
		return fmt.Errorf("failed to marshal enrollment: %v", err)
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(enrollmentJSON))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %v", err)
	}
	req.Header.Set("Authorization", sasToken)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create enrollment: %v", string(body))
	}

	return nil
}

// deleteEnrollment deletes an enrollment from Azure IoT DPS.
func deleteEnrollment(enrollmentID, provService, sasToken string) error {
	url := fmt.Sprintf(enrollmentAPIURL, provService, enrollmentID)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %v", err)
	}
	req.Header.Set("Authorization", sasToken)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete enrollment: %v", string(body))
	}

	return nil
}

func readPublicKey(handle tpmutil.Handle) ([]byte, error) {
	// unfortunately we can't used SWTPM socket directly, it is blocked because
	// qemu is using it, so we have to use ssh and tpm2-tools
	tpmToolsPath := "/containers/services/vtpm/lower/usr/bin/tpm2"
	tpmToolsLibPath := "/containers/services/vtpm/lower/usr/lib"
	command := fmt.Sprintf("LD_LIBRARY_PATH=%s %s readpublic -Q -c 0x%x -o pub.pub", tpmToolsLibPath, tpmToolsPath, handle)
	_, err := eveNode.EveRunCommand(command)
	if err != nil {
		return nil, err
	}

	out, err := eveNode.EveReadFile("pub.pub")
	if err != nil {
		return nil, err
	}

	err = eveNode.EveDeleteFile("pub.pub")
	if err != nil {
		return nil, err
	}

	return out, nil
}

func readEveEndorsmentKey() (string, string, error) {
	pub, err := readPublicKey(etpm.TpmEKHdl)
	if err != nil {
		return "", "", err
	}

	hash := sha256.Sum256(pub)
	hashHex := hex.EncodeToString(hash[:])

	return base64.StdEncoding.EncodeToString(pub), hashHex, nil
}

func getAzureIoTServicesStatus(output string) (map[string]string, error) {
	// this what is being parse:
	//$ sudo iotedge system status
	//System services:
	//aziot-edged             Running
	//aziot-identityd         Down - activating
	//aziot-keyd              Ready
	//aziot-certd             Ready
	//aziot-tpmd              Running

	// Flag to indicate if we are in the "System services" section
	inSystemServices := false
	services := make(map[string]string, 0)

	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()

		// Detect when we are in the "System services" section
		if strings.Contains(line, "System services:") {
			inSystemServices = true
			continue
		}

		// Exit the loop when we are out of the "System services" section
		if inSystemServices && strings.TrimSpace(line) == "" {
			break
		}

		if inSystemServices {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				services[parts[0]] = strings.Join(parts[1:], " ")
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return services, nil
}
