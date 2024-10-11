package aziot

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tk "github.com/lf-edge/eden/pkg/evetestkit"
	log "github.com/sirupsen/logrus"
)

const (
	sshPort            = "8027"
	testScriptBasePath = "/home/ubuntu/"
	projectName        = "aziot-test"
	aziotwait          = 30      // seconds
	appWait            = 60 * 10 // 10 minutes
	sshWait            = 60 * 5  // 5 minutes
	nodeRebootWait     = 60 * 5  // 5 minutes
)

var (
	appLink    = "https://cloud-images.ubuntu.com/releases/22.04/release/ubuntu-22.04-server-cloudimg-amd64.img"
	testScript = "scripts/test_ubuntu22.04_aziot_latest.sh"
	eveNode    *tk.EveNode
	// We need a shared access policy with the following permissions:
	// Registration Status Read, Registration Status Write, Enrollment Read, Enrollment Write
	// We can create a new policy in the Azure portal by going to :
	// IoT Hub -> Device Provisioning Service (DPS) -> Shared access policies -> Add
	// and then copy the connection string.
	connectionString = os.Getenv("AZIOT_CONNECTION_STRING")
	// The ID Scope is required to configure azure-iot in the VM,
	// we can get it from the Azure IoT Hub -> Device Provisioning Service -> Overview
	// and copy the "ID Scope".
	aziotIDScope = os.Getenv("AZIOT_ID_SCOPE")
	appName      = ""
)

func deleteApp(appName string) {
	err := eveNode.AppStopAndRemove(appName)
	if err != nil {
		log.Errorf("Failed to stop and remove app: %v", err)
	}
}

func setupApp() (string, error) {
	appName := tk.GetRandomAppName(projectName + "-")
	pubPorts := []string{sshPort + ":22"}
	pc := tk.GetDefaultVMConfig(appName, tk.AppDefaultCloudConfig, pubPorts)
	err := eveNode.EveDeployApp(appLink, pc,
		tk.WithSSH(tk.AppDefaultSSHUser, tk.AppDefaultSSHPass, sshPort))
	if err != nil {
		return "", fmt.Errorf("failed to deploy app: %v", err)
	}

	return appName, nil
}

func waitForApp(appName string) error {
	// Wait for the app to start and ssh to be ready
	log.Printf("Waiting for app %s to start...", appName)
	err := eveNode.AppWaitForRunningState(appName, appWait)
	if err != nil {
		return fmt.Errorf("failed to wait for app to start: %v", err)
	}
	log.Printf("Waiting for ssh to be ready...")
	err = eveNode.AppWaitForSSH(appName, sshWait)
	if err != nil {
		return fmt.Errorf("failed to wait for ssh: %v", err)
	}

	log.Println("SSH connection established")
	return nil
}

func checkAziotServices(t *testing.T, appName string) {
	// Check the status of the iotedge services
	status, err := eveNode.AppSSHExec(appName, "sudo iotedge system status")
	if err != nil {
		t.Fatalf("Failed to get iotedge status: %v", err)
	}
	services, err := getAzureIoTServicesStatus(status)
	if err != nil {
		t.Fatalf("Failed to get Azure IoT services status: %v", err)
	}

	// If all services are running we are good, otherwise fail the test
	for service, status := range services {
		if strings.ToLower(status) != "running" {
			// Errorf calls Fail(), so we don't need to call it explicitly
			t.Errorf("Service %s is not running", service)
		}
	}
	t.Log("====================== SERVICES STATUS ======================")
	for service, status := range services {
		t.Logf("%s: \t\t%s\n", service, status)
	}

	if t.Failed() {
		// Get the aziot-tpmd logs, in one test we patch this service with eve-tools
		// so good to have the logs for debugging.
		command := "sudo iotedge system logs | grep aziot-tpmd"
		tpmLog, err := eveNode.AppSSHExec(appName, command)
		if err != nil {
			t.Errorf("Failed to get aziot-tpmd logs: %v", err)
		} else {
			t.Log("====================== TPMD LOG ======================")
			t.Log(tpmLog)
		}

		// Get all the errors from the aziot logs
		command = "sudo iotedge system logs | grep ERR | sed 's/.*ERR!] - //' | sort | uniq"
		errors, err := eveNode.AppSSHExec(appName, command)
		if err != nil {
			t.Errorf("Failed to error logs: %v", err)
		} else {
			t.Log("====================== ERRORS ======================")
			t.Log(errors)
		}
	}
}

func TestMain(m *testing.M) {
	log.Println("vTPM Test started")
	defer log.Println("vTPM Test finished")

	node, err := tk.InitilizeTest(projectName, tk.WithControllerVerbosity("debug"))
	if err != nil {
		log.Fatalf("Failed to initialize test: %v", err)
	}

	eveNode = node
	name, err := setupApp()
	if err != nil {
		log.Fatalf("Failed to setup app: %v", err)
	}

	// Wait for the deployed app to appear in the list
	time.Sleep(30 * time.Second)
	appName = name

	err = waitForApp(name)
	if err != nil {
		deleteApp(name)
		log.Fatalf("Failed to wait for app: %v", err)
	}

	res := m.Run()
	deleteApp(name)
	os.Exit(res)
}

// TestVtpmIsRunningOnEVE checks if the vTPM process is running on the EVE node,
// it does this by checking if the vTPM control socket is open and the vTPM process
// is listening on it.
func TestVtpmIsRunningOnEVE(t *testing.T) {
	t.Log("TestVtpmIsRunningOnEVE started")
	defer t.Log("TestVtpmIsRunningOnEVE finished")

	// find the vTPM control socket and see if the vTPM process is listening on it.
	command := "lsof -U | grep $(cat /proc/net/unix | grep vtpm | awk '{print $7}')"
	out, err := eveNode.EveRunCommand(command)
	if err != nil {
		t.Fatalf("Failed to check if vTPM is running on EVE: %v", err)
	}

	if len(out) == 0 || !strings.Contains(string(out), "vtpm") {
		t.Fatalf("vTPM is not running on EVE : %s", out)
	}
}

// TestVtpmIsStatePreservation checks if the vTPM state is preserved after a reboot,
// it does this by creating a key in the vTPM (through a VM running on EVE) and
// then rebooting the EVE node, after the reboot it checks if the key is still
// present in the vTPM, by getting the list of vTPM persistent keys (through the
// the VM running on EVE).
func TestVtpmIsStatePreservation(t *testing.T) {
	t.Log("TestVtpmIsStatePreservation started")
	defer t.Log("TestVtpmIsStatePreservation finished")

	t.Log("Copying the key creation script to the VM")
	createKeyScriptPath := testScriptBasePath + "test_make_tpm_keys.sh"
	err := eveNode.AppSCPCopy(appName, "scripts/test_make_tpm_keys.sh", createKeyScriptPath)
	if err != nil {
		t.Fatalf("Failed to copy file to vm: %v", err)
	}

	// Prepare the script for execution
	command := fmt.Sprintf("chmod +x %s", createKeyScriptPath)
	out, err := eveNode.AppSSHExec(appName, command)
	if err != nil {
		t.Fatalf("Failed perpare the TPM key creation script for execution \"%s\" : %v", out, err)
	}

	// Execute the script to create the necessary TPM keys
	_, err = eveNode.AppSSHExec(appName, createKeyScriptPath)
	if err != nil {
		t.Fatalf("Failed to execute TPM key creation script in VM: %v", err)
	}

	// Reboot the EVE node and check the vTPM state is preserved
	err = eveNode.EveRebootAndWait(nodeRebootWait)
	if err != nil {
		t.Fatalf("Failed to reboot EVE: %v", err)
	}

	err = waitForApp(appName)
	if err != nil {
		t.Fatalf("Failed to wait for app: %v", err)
	}

	createStatePresScriptPath := testScriptBasePath + "test_vtpm_state_preservation.sh"
	err = eveNode.AppSCPCopy(appName, "scripts/test_vtpm_state_preservation.sh", createStatePresScriptPath)
	if err != nil {
		t.Fatalf("Failed to copy file to vm: %v", err)
	}

	// Prepare the script for execution
	command = fmt.Sprintf("chmod +x %s", createStatePresScriptPath)
	_, err = eveNode.AppSSHExec(appName, command)
	if err != nil {
		t.Fatalf("Failed perpare the TPM state preservation test script for execution: %v", err)
	}
}

// TestAzureIotTPMEndrolmentWithVTPM tests the end-to-end scenario of enrolling a TPM device
// in Azure IoT Hub, this test will create a TPM enrollment in Azure IoT Hub, configure the
// Azure IoT Edge in a VM running on EVE, and check if the services are running.
func TestAzureIotTPMEndrolmentWithVTPM(t *testing.T) {
	t.Log("TestAzureIotTPMEndrolmentWithTPM started")
	t.Log("Setup :\n\tAziot (latest) on Ubuntu-22.04-amd64\n\twith VTPM")
	defer t.Log("TestAzureIotTPMEndrolmentWithVTPM finished")

	if !eveNode.EveIsTpmEnabled() {
		t.Skip("TPM is not enabled, skipping test")
	}

	// Check for secrets, if not available don't bother running the tests.
	if connectionString == "" {
		log.Fatalf("AZIOT_CONNECTION_STRING environment variable is not set")
	}
	if aziotIDScope == "" {
		log.Fatalf("AZIOT_ID_SCOPE environment variable is not set")
	}

	// Copy the test script to the VM
	testScriptPath := testScriptBasePath + filepath.Base(testScript)
	err := eveNode.AppSCPCopy(appName, testScript, testScriptPath)
	if err != nil {
		t.Fatalf("Failed to copy file to vm: %v", err)
	}
	t.Log("Test script copied to VM")

	// for this to test to work, we need to create an enrollment in the Azure IoT Hub.
	t.Log("Creating a TPM enrollment in Azure IoT Hub")
	createKeyScriptPath := testScriptBasePath + "test_make_tpm_keys.sh"
	err = eveNode.AppSCPCopy(appName, "scripts/test_make_tpm_keys.sh", createKeyScriptPath)
	if err != nil {
		t.Fatalf("Failed to copy file to vm: %v", err)
	}

	// Prepare the script for execution
	command := fmt.Sprintf("chmod +x %s", createKeyScriptPath)
	_, err = eveNode.AppSSHExec(appName, command)
	if err != nil {
		t.Fatalf("Failed perpare the TPM key creation script for execution: %v", err)
	}

	// Execute the script to create the necessary TPM keys
	out, err := eveNode.AppSSHExec(appName, createKeyScriptPath)
	if err != nil {
		t.Fatalf("Failed to execute TPM key creation script in VM \"%s\": %v", out, err)
	}

	// Get the endorsement key from the VM
	ek, err := eveNode.AppSSHExec(appName, "base64 -w0 ek.pub")
	if err != nil {
		t.Fatalf("Failed to read endrosment key from VM: %v", err)
	}

	// Get the enrollment ID from the VM
	command = "sha256sum -b ek.pub | cut -d' ' -f1 | sed -e 's/[^[:alnum:]]//g'"
	id, err := eveNode.AppSSHExec(appName, command)
	if err != nil {
		t.Fatalf("Failed to get enrollment ID from VM: %v", err)
	}
	endorsementKey, enrollmentID := strings.TrimSpace(ek), strings.TrimSpace(id)

	// Get the provisioning service name from the connection string
	provService, err := getProvisioningService(connectionString)
	if err != nil {
		t.Fatalf("Failed to get provisioning service: %v\n", err)
	}

	// From the connection string generate a SAS token lasting for 1 hour
	sasToken, err := getSasTokenFromConnectionString(connectionString, 1)
	if err != nil {
		t.Fatalf("Failed to generate a SAS token: %v\n", err)
	}

	// Add the enrollment to azure iot hub portal
	err = addTPMEnrollment(enrollmentID, endorsementKey, provService, sasToken)
	if err != nil {
		t.Fatalf("Failed to create a TPM enrollment on Azure: %v\n", err)
	}
	defer func() {
		err = deleteEnrollment(enrollmentID, provService, sasToken)
		if err != nil {
			log.Printf("Failed to delete TPM enrollment, please remove it manually: %v\n", err)
		}
	}()

	// Prepare the test script for execution
	command = fmt.Sprintf("chmod +x %s", testScriptPath)
	_, err = eveNode.AppSSHExec(appName, command)
	if err != nil {
		t.Fatalf("Failed perpare the test script for execution: %v", err)
	}

	// Execute the test script, this will configure the azure-iot in the VM
	// and start the services.
	command = fmt.Sprintf("ID_SCOPE=%s REGISTRATION_ID=%s %s", aziotIDScope, enrollmentID, testScriptPath)
	_, err = eveNode.AppSSHExec(appName, command)
	if err != nil {
		t.Fatalf("Failed to execute test script in VM: %v", err)
	}

	// Wait for the services to start
	t.Logf("Waiting for services to start...")
	time.Sleep(aziotwait * time.Second)
	// this will check the status of the services and fail the test if any service is not running
	checkAziotServices(t, appName)
}
