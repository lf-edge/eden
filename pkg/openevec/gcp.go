package openevec

import (
	"fmt"

	"github.com/lf-edge/eden/pkg/linuxkit"
)

func GcpImageDelete(gcpKey, gcpProjectName, gcpImageName, gcpBucketName string) error {
	gcpClient, err := linuxkit.NewGCPClient(gcpKey, gcpProjectName)
	if err != nil {
		return fmt.Errorf("unable to connect to GCP: %v", err)
	}
	fileName := fmt.Sprintf("%s.img.tar.gz", gcpImageName)
	err = gcpClient.DeleteImage(gcpImageName)
	if err != nil {
		return fmt.Errorf("error in delete of Google Compute Image: %v", err)
	}
	if err := gcpClient.RemoveFile(fileName, gcpBucketName); err != nil {
		return fmt.Errorf("error id delete from Google Storage: %v", err)
	}
	return nil
}

func GcpImageUpload(gcpKey, gcpProjectName, gcpImageName, gcpBucketName, eveImageFile string, gcpvTPM bool) error {
	gcpClient, err := linuxkit.NewGCPClient(gcpKey, gcpProjectName)
	if err != nil {
		return fmt.Errorf("Unable to connect to GCP: %v", err)
	}
	fileName := fmt.Sprintf("%s.img.tar.gz", gcpImageName)
	if err := gcpClient.UploadFile(eveImageFile, fileName, gcpBucketName, false); err != nil {
		return fmt.Errorf("Error copying to Google Storage: %v", err)
	}
	err = gcpClient.CreateImage(gcpImageName, "https://storage.googleapis.com/"+gcpBucketName+"/"+fileName, "", gcpvTPM, true)
	if err != nil {
		return fmt.Errorf("Error creating Google Compute Image: %v", err)
	}

	return nil
}

func GcpRun(gcpKey, gcpProjectName, gcpImageName, gcpVMName, gcpZone, gcpMachineType string, gcpvTPM bool, eveDisks, eveImageSizeMB int) error {
	gcpClient, err := linuxkit.NewGCPClient(gcpKey, gcpProjectName)
	if err != nil {
		return fmt.Errorf("Unable to connect to GCP: %v", err)
	}
	disks := linuxkit.Disks{}
	for i := 0; i < eveDisks; i++ {
		disks = append(disks, linuxkit.DiskConfig{Size: eveImageSizeMB})
	}
	if err := gcpClient.CreateInstance(gcpVMName, gcpImageName, gcpZone, gcpMachineType, disks, nil, gcpvTPM, true); err != nil {
		return fmt.Errorf("CreateInstance: %s", err)
	}

	return nil
}

func GcpDelete(gcpKey, gcpProjectName, gcpVMName, gcpZone string) error {
	gcpClient, err := linuxkit.NewGCPClient(gcpKey, gcpProjectName)
	if err != nil {
		return fmt.Errorf("Unable to connect to GCP: %v", err)
	}
	if err := gcpClient.DeleteInstance(gcpVMName, gcpZone, true); err != nil {
		return fmt.Errorf("DeleteInstance: %s", err)
	}

	return nil
}
