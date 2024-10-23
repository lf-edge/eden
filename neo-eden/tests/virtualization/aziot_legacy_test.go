package virtualization

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bloomberg/go-testgroup"
	tk "github.com/lf-edge/eden/pkg/evetestkit"
)

var (
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

func (grp *VirtualizationTests) TestAzureIotTPMEndrolmentWithEveTools(t *testgroup.T) {
	eveNode.LogTimeInfof("TestAzureIotTPMEndrolmentWithEveTools started")
	eveNode.LogTimeInfof("Setup :\n\tAziot version 1.4.0 on Ubuntu-20.04-amd64\n\twith EVE-Tools and PTPM")
	defer eveNode.LogTimeInfof("TestAzureIotTPMEndrolmentWithEveTools finished")

	// Check for secrets, if not available don't bother running the tests.
	if connectionString == "" {
		eveNode.LogTimeFatalf("AZIOT_CONNECTION_STRING environment variable is not set")
	}
	if aziotIDScope == "" {
		eveNode.LogTimeFatalf("AZIOT_ID_SCOPE environment variable is not set")
	}

	appName := tk.GetRandomAppName(projectName + "-")
	appName, err := eveNode.EveDeployUbuntu(tk.Ubuntu2004, appName, false)
	if err != nil {
		eveNode.LogTimeFatalf("Failed to deploy app: %v", err)
	}
	defer func() {
		err = eveNode.AppStopAndRemove(appName)
		if err != nil {
			eveNode.LogTimeFatalf("Failed to remove app %s: %v", appName, err)
		}
	}()

	err = waitForApp(appName, appWait, sshWait)
	if err != nil {
		eveNode.LogTimeFatalf("Failed to wait for app: %v", err)
	}

	eveNode.LogTimeInfof("Copying test scripts to the vm...")
	err = eveNode.CopyTestScripts(appName, testScriptBasePath, &azureIoTTestScripts)
	if err != nil {
		eveNode.LogTimeFatalf("Failed to copy test scripts to the vm: %v", err)
	}

	// for this to test to work, we need to create an enrollment in the Azure IoT Hub,
	// the enrolment should be created with the endorsement key of the TPM and
	// since we are running EVE in QEMU with SWTPM, the endorsement key changes
	// every time we start the VM EVE, so we need to read it, create the enrollment,
	// run the test and delete the enrollment.
	eveNode.LogTimeInfof("Creating a TPM enrollment in Azure IoT Hub")
	endorsementKey, enrollmentID := "", ""

	// read the endorsement key from the EVE.
	ek, id, err := readEveEndorsmentKey()
	if err != nil {
		eveNode.LogTimeFatalf("Failed to read endorsement key from EVE: %v", err)
	}
	endorsementKey, enrollmentID = ek, id

	// Get the provisioning service name from the connection string
	provService, err := getProvisioningService(connectionString)
	if err != nil {
		eveNode.LogTimeFatalf("Failed to get provisioning service: %v\n", err)
	}

	// From the connection string generate a SAS token lasting for 1 hour
	sasToken, err := getSasTokenFromConnectionString(connectionString, 1)
	if err != nil {
		eveNode.LogTimeFatalf("Failed to generate a SAS token: %v\n", err)
	}

	// Add the enrollment to azure iot hub portal
	err = addTPMEnrollment(enrollmentID, endorsementKey, provService, sasToken)
	if err != nil {
		eveNode.LogTimeFatalf("Failed to create a TPM enrollment on Azure: %v\n", err)
	}
	defer func() {
		err = deleteEnrollment(enrollmentID, provService, sasToken)
		if err != nil {
			eveNode.LogTimeInfof("Failed to delete TPM enrollment, please remove it manually: %v\n", err)
		}
	}()

	// Execute the test script, this will configure the azure-iot in the VM
	// and start the services.
	command := fmt.Sprintf("ID_SCOPE=%s REGISTRATION_ID=%s %s", aziotIDScope, enrollmentID, eveNode.GetCopiedScriptPath("test_aziot_1.4.0.sh"))
	_, err = eveNode.AppSSHExec(appName, command)
	if err != nil {
		eveNode.LogTimeFatalf("Failed to execute test script in VM: %v", err)
	}

	// Wait for the services to start
	eveNode.LogTimeInfof("Waiting for services to start...")
	time.Sleep(aziotwait * time.Second)

	// this will check the status of the services and fail the test if any service is not running
	checkAziotServices(appName, t)

	eveNode.LogTimeInfof("TestAzureIotTPMEndrolmentWithEveTools passed")
}

func checkAziotServices(appName string, t *testgroup.T) {
	// Check the status of the iotedge services
	status, err := eveNode.AppSSHExec(appName, "sudo iotedge system status")
	if err != nil {
		eveNode.LogTimeFatalf("Failed to get iotedge status: %v", err)
	}
	services, err := getAzureIoTServicesStatus(status)
	if err != nil {
		eveNode.LogTimeFatalf("Failed to get Azure IoT services status: %v", err)
	}

	// If all services are running we are good, otherwise fail the test
	for service, status := range services {
		if strings.ToLower(status) != "running" {
			// Errorf calls Fail(), so we don't need to call it explicitly
			eveNode.LogTimeErrorf("Service %s is not running", service)
		}
	}
	eveNode.LogTimeInfof("====================== SERVICES STATUS ======================")
	for service, status := range services {
		eveNode.LogTimeInfof("%s: \t\t%s\n", service, status)
	}

	if t.Failed() {
		// Get the aziot-tpmd logs, in one test we patch this service with eve-tools
		// so good to have the logs for debugging.
		command := "sudo iotedge system logs | grep aziot-tpmd"
		tpmLog, err := eveNode.AppSSHExec(appName, command)
		if err != nil {
			eveNode.LogTimeErrorf("Failed to get aziot-tpmd logs: %v", err)
		} else {
			eveNode.LogTimeInfof("====================== TPMD LOG ======================")
			eveNode.LogTimeInfof(tpmLog)
		}

		// Get all the errors from the aziot logs
		command = "sudo iotedge system logs | grep ERR | sed 's/.*ERR!] - //' | sort | uniq"
		errors, err := eveNode.AppSSHExec(appName, command)
		if err != nil {
			eveNode.LogTimeErrorf("Failed to error logs: %v", err)
		} else {
			eveNode.LogTimeInfof("====================== ERRORS ======================")
			eveNode.LogTimeInfof(errors)
		}
	}
}
