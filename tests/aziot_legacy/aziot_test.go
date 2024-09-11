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
)

var (
	appLink    = "https://cloud-images.ubuntu.com/releases/20.04/release/ubuntu-20.04-server-cloudimg-amd64.img"
	testScript = "scripts/test_ubuntu20.04_aziot_1.4.0.sh"
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
)

func TestMain(m *testing.M) {
	log.Println("Azure IOT Hub Legacy Test started")
	defer log.Println("Azure IOT Hub Legacy Test finished")

	// Check for secrets, if not available don't bother running the tests.
	if connectionString == "" {
		log.Fatalf("AZIOT_CONNECTION_STRING environment variable is not set")
	}
	if aziotIDScope == "" {
		log.Fatalf("AZIOT_ID_SCOPE environment variable is not set")
	}

	node, err := tk.InitilizeTest(projectName, tk.WithControllerVerbosity("debug"))
	if err != nil {
		log.Fatalf("Failed to initialize test: %v", err)
	}

	eveNode = node
	res := m.Run()
	os.Exit(res)
}

func TestAzureIotTPMEndrolmentWithEveTools(t *testing.T) {
	t.Log("TestAzureIotTPMEndrolmentWithEveTools started")
	t.Log("Setup :\n\tAziot version 1.4.0 on Ubuntu-20.04-amd64\n\twith EVE-Tools and PTPM")
	defer t.Log("TestAzureIotTPMEndrolmentWithEveTools finished")

	if !eveNode.EveIsTpmEnabled() {
		t.Skip("TPM is not enabled, skipping test")
	}

	testAzureIotEdge(t)
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

func waitForApp(t *testing.T, appName string) error {
	// Wait for the app to start and ssh to be ready
	t.Logf("Waiting for app %s to start...", appName)
	err := eveNode.AppWaitForRunningState(appName, appWait)
	if err != nil {
		return fmt.Errorf("failed to wait for app to start: %v", err)
	}
	t.Logf("Waiting for ssh to be ready...")
	err = eveNode.AppWaitForSSH(appName, sshWait)
	if err != nil {
		return fmt.Errorf("failed to wait for ssh: %v", err)
	}

	t.Log("SSH connection established")
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

func testAzureIotEdge(t *testing.T) {
	appName, err := setupApp()
	if err != nil {
		t.Fatalf("Failed to setup app: %v", err)
	}
	defer func() {
		err = eveNode.AppStopAndRemove(appName)
		if err != nil {
			log.Errorf("Failed to stop and remove app: %v", err)
		}
	}()

	// Wait for the deployed app to appear in the list
	time.Sleep(30 * time.Second)
	err = waitForApp(t, appName)
	if err != nil {
		t.Fatalf("Failed to wait for app: %v", err)
	}

	// Copy the test script to the VM
	testScriptPath := testScriptBasePath + filepath.Base(testScript)
	err = eveNode.AppSCPCopy(appName, testScript, testScriptPath)
	if err != nil {
		t.Fatalf("Failed to copy file to vm: %v", err)
	}
	t.Log("Test script copied to VM")

	// for this to test to work, we need to create an enrollment in the Azure IoT Hub,
	// the enrolment should be created with the endorsement key of the TPM and
	// since we are running EVE in QEMU with SWTPM, the endorsement key changes
	// every time we start the VM EVE, so we need to read it, create the enrollment,
	// run the test and delete the enrollment.
	t.Log("Creating a TPM enrollment in Azure IoT Hub")
	endorsementKey, enrollmentID := "", ""

	// read the endorsement key from the EVE.
	ek, id, err := readEveEndorsmentKey()
	if err != nil {
		t.Errorf("Failed to read endorsement key from EVE: %v", err)
	}
	endorsementKey, enrollmentID = ek, id

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
	command := fmt.Sprintf("chmod +x %s", testScriptPath)
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
