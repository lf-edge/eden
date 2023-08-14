package linuxkit

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/moby/term"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/api/storage/v1"
)

const pollingInterval = time.Second
const timeout = 60
const nested = true

// GCPClient contains state required for communication with GCP
type GCPClient struct {
	compute     *compute.Service
	storage     *storage.Service
	projectName string
	privKey     *rsa.PrivateKey
}

// NewGCPClient creates a new GCP client
func NewGCPClient(keys, projectName string) (*GCPClient, error) {
	log.Debugf("Connecting to GCP")
	ctx := context.Background()
	var client *GCPClient
	if projectName == "" {
		return nil, fmt.Errorf("the project name is not specified")
	}
	var opts []option.ClientOption
	if keys != "" {
		log.Debugf("Using Keys %s", keys)
		f, err := os.Open(keys)
		if err != nil {
			return nil, err
		}

		jsonKey, err := io.ReadAll(f)
		if err != nil {
			return nil, err
		}

		config, err := google.JWTConfigFromJSON(jsonKey,
			storage.DevstorageReadWriteScope,
			compute.ComputeScope,
		)
		if err != nil {
			return nil, err
		}

		opts = append(opts, option.WithAPIKey(string(jsonKey)))
		opts = append(opts, option.WithHTTPClient(config.Client(ctx)))
		client = &GCPClient{
			projectName: projectName,
		}
	} else {
		log.Debugf("Using Application Default credentials")
		gc, err := google.DefaultClient(
			ctx,
			storage.DevstorageReadWriteScope,
			compute.ComputeScope,
		)
		if err != nil {
			return nil, err
		}
		opts = append(opts, option.WithHTTPClient(gc))
		client = &GCPClient{
			projectName: projectName,
		}
	}

	var err error

	client.compute, err = compute.NewService(ctx, opts...)
	if err != nil {
		return nil, err
	}

	client.storage, err = storage.NewService(ctx, opts...)
	if err != nil {
		return nil, err
	}

	log.Debugf("Generating SSH Keypair")
	client.privKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// UploadFile uploads a file to Google Storage
func (g GCPClient) UploadFile(src, dst, bucketName string, public bool) error {
	log.Infof("Uploading file %s to Google Storage as %s", src, dst)
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	objectCall := g.storage.Objects.Insert(bucketName, &storage.Object{Name: dst}).Media(f)

	if public {
		objectCall.PredefinedAcl("publicRead")
	}

	_, err = objectCall.Do()
	if err != nil {
		return err
	}
	log.Infof("Upload Complete!")
	fmt.Println("gs://" + bucketName + "/" + dst)
	return nil
}

// RemoveFile removes a file from Google Storage
func (g GCPClient) RemoveFile(file, bucketName string) error {
	log.Infof("Removing of file %s from Google Storage", file)

	if err := g.storage.Objects.Delete(bucketName, file).Do(); err != nil {
		return err
	}
	log.Infof("Removing Complete!")
	return nil
}

// CreateImage creates a GCP image using the source from Google Storage
func (g GCPClient) CreateImage(name, storageURL, family string, uefi, replace bool) error {
	if replace {
		if err := g.DeleteImage(name); err != nil {
			return err
		}
	}

	log.Infof("Creating image: %s", name)
	imgObj := &compute.Image{
		RawDisk: &compute.ImageRawDisk{
			Source: storageURL,
		},
		Name: name,
	}

	if family != "" {
		imgObj.Family = family
	}

	if nested {
		imgObj.Licenses = []string{"projects/vm-options/global/licenses/enable-vmx"}
	}

	if uefi {
		imgObj.GuestOsFeatures = []*compute.GuestOsFeature{
			{Type: "UEFI_COMPATIBLE"},
		}
	}

	op, err := g.compute.Images.Insert(g.projectName, imgObj).Do()
	if err != nil {
		return err
	}

	if err := g.pollOperationStatus(op.Name); err != nil {
		return err
	}
	log.Infof("Image %s created", name)
	return nil
}

// DeleteImage deletes and image
func (g GCPClient) DeleteImage(name string) error {
	var notFound bool
	op, err := g.compute.Images.Delete(g.projectName, name).Do()
	if err != nil {
		if _, ok := err.(*googleapi.Error); !ok {
			return err
		}
		if err.(*googleapi.Error).Code != 404 {
			return err
		}
		notFound = true
	}
	if !notFound {
		log.Infof("Deleting existing image...")
		if err := g.pollOperationStatus(op.Name); err != nil {
			return err
		}
		log.Infof("Image %s deleted", name)
	}
	return nil
}

// ListImages list all uploaded images
func (g GCPClient) ListImages() ([]string, error) {
	var result []string
	var notFound bool
	op, err := g.compute.Images.List(g.projectName).Do()
	if err != nil {
		if _, ok := err.(*googleapi.Error); !ok {
			return nil, err
		}
		if err.(*googleapi.Error).Code != 404 {
			return nil, err
		}
		notFound = true
	}
	if op == nil {
		return result, nil
	}
	if !notFound {
		for _, el := range op.Items {
			result = append(result, el.Name)
		}
	}
	return result, nil
}

// CreateInstance creates and starts an instance on GCP
func (g GCPClient) CreateInstance(name, image, zone, machineType string, disks Disks, data *string, vtpm, replace bool) error {
	if replace {
		if err := g.DeleteInstance(name, zone, true); err != nil {
			return err
		}
	}

	log.Infof("Creating instance %s from image %s (type: %s in %s)", name, image, machineType, zone)

	enabled := new(string)
	*enabled = "1"

	k, err := ssh.NewPublicKey(g.privKey.Public())
	if err != nil {
		return err
	}
	sshKey := new(string)
	*sshKey = fmt.Sprintf("moby:%s moby", string(ssh.MarshalAuthorizedKey(k)))

	instanceDisks := []*compute.AttachedDisk{
		{
			AutoDelete: true,
			Boot:       true,
			InitializeParams: &compute.AttachedDiskInitializeParams{
				SourceImage: fmt.Sprintf("global/images/%s", image),
			},
		},
	}

	for i, disk := range disks {
		var diskName string
		if disk.Path != "" {
			diskName = disk.Path
		} else {
			diskName = fmt.Sprintf("%s-disk-%d", name, i)
		}
		var diskSizeGb int64
		if disk.Size == 0 {
			diskSizeGb = int64(1)
		} else {
			diskSizeGb = int64(convertMBtoGB(disk.Size))
		}
		disk := &compute.Disk{Name: diskName, SizeGb: diskSizeGb}
		if vtpm {
			disk.GuestOsFeatures = []*compute.GuestOsFeature{
				{Type: "UEFI_COMPATIBLE"},
			}
		}
		diskOp, err := g.compute.Disks.Insert(g.projectName, zone, disk).Do()
		if err != nil {
			return err
		}
		if err != nil {
			return err
		}
		if err := g.pollZoneOperationStatus(diskOp.Name, zone); err != nil {
			return err
		}
		instanceDisks = append(instanceDisks, &compute.AttachedDisk{
			AutoDelete: true,
			Boot:       false,
			Source:     fmt.Sprintf("zones/%s/disks/%s", zone, diskName),
		})
	}

	instanceObj := &compute.Instance{
		MachineType: fmt.Sprintf("zones/%s/machineTypes/%s", zone, machineType),
		Name:        name,
		Disks:       instanceDisks,
		NetworkInterfaces: []*compute.NetworkInterface{
			{
				Network: "global/networks/default",
				AccessConfigs: []*compute.AccessConfig{
					{
						Type: "ONE_TO_ONE_NAT",
					},
				},
			},
		},
		Metadata: &compute.Metadata{
			Items: []*compute.MetadataItems{
				{
					Key:   "serial-port-enable",
					Value: enabled,
				},
				{
					Key:   "ssh-keys",
					Value: sshKey,
				},
				{
					Key:   "user-data",
					Value: data,
				},
			},
		},
	}
	if nested {
		// TODO(rn): We could/should check here if the image has nested virt enabled
		instanceObj.MinCpuPlatform = "Intel Skylake"
	}
	if vtpm {
		instanceObj.ShieldedInstanceConfig = &compute.ShieldedInstanceConfig{EnableVtpm: true}
	}
	op, err := g.compute.Instances.Insert(g.projectName, zone, instanceObj).Do()
	if err != nil {
		return err
	}
	if err := g.pollZoneOperationStatus(op.Name, zone); err != nil {
		return err
	}
	log.Infof("Instance created")
	return nil
}

// DeleteInstance removes an instance
func (g GCPClient) DeleteInstance(instance, zone string, wait bool) error {
	var notFound bool
	op, err := g.compute.Instances.Delete(g.projectName, zone, instance).Do()
	if err != nil {
		if _, ok := err.(*googleapi.Error); !ok {
			return err
		}
		if err.(*googleapi.Error).Code != 404 {
			return err
		}
		notFound = true
	}
	if !notFound && wait {
		log.Infof("Deleting existing instance...")
		if err := g.pollZoneOperationStatus(op.Name, zone); err != nil {
			return err
		}
		log.Infof("Instance %s deleted", instance)
	}
	return nil
}

// GetInstanceSerialOutput streams the serial output of an instance
// follow log if follow set to true
func (g GCPClient) GetInstanceSerialOutput(instance, zone string, follow bool) error {
	log.Infof("Getting serial port output for instance %s", instance)
	var next int64
	for {
		res, err := g.compute.Instances.GetSerialPortOutput(g.projectName, zone, instance).Start(next).Do()
		if err != nil {
			if err.(*googleapi.Error).Code == 400 {
				// Instance may not be ready yet...
				time.Sleep(pollingInterval)
				continue
			}
			if err.(*googleapi.Error).Code == 503 {
				// Timeout received when the instance has terminated
				break
			}
			return err
		}
		fmt.Printf(res.Contents)
		if !follow {
			break
		}
		next = res.Next
		// When the instance has been stopped, Start and Next will both be 0
		if res.Start > 0 && next == 0 {
			break
		}
	}
	return nil
}

// ConnectToInstanceSerialPort uses SSH to connect to the serial port of the instance
func (g GCPClient) ConnectToInstanceSerialPort(instance, zone string) error {
	log.Infof("Connecting to serial port of instance %s", instance)
	gPubKeyURL := "https://cloud-certs.storage.googleapis.com/google-cloud-serialport-host-key.pub"
	resp, err := http.Get(gPubKeyURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	gPubKey, _, _, _, err := ssh.ParseAuthorizedKey(body)
	if err != nil {
		return err
	}

	signer, err := ssh.NewSignerFromKey(g.privKey)
	if err != nil {
		return err
	}
	config := &ssh.ClientConfig{
		User: fmt.Sprintf("%s.%s.%s.moby", g.projectName, zone, instance),
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.FixedHostKey(gPubKey),
		Timeout:         5 * time.Second,
	}

	var conn *ssh.Client
	// Retry connection as VM may not be ready yet
	for i := 0; i < timeout; i++ {
		conn, err = ssh.Dial("tcp", "ssh-serialport.googleapis.com:9600", config)
		if err != nil {
			time.Sleep(pollingInterval)
			continue
		}
		break
	}
	if conn == nil {
		return fmt.Errorf(err.Error())
	}
	defer conn.Close()

	session, err := conn.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("Unable to setup stdin for session: %v", err)
	}
	go func() {
		_, _ = io.Copy(stdin, os.Stdin)
	}()

	stdout, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("Unable to setup stdout for session: %v", err)
	}
	go func() {
		_, _ = io.Copy(os.Stdout, stdout)
	}()

	stderr, err := session.StderrPipe()
	if err != nil {
		return fmt.Errorf("Unable to setup stderr for session: %v", err)
	}
	go func() {
		_, _ = io.Copy(os.Stderr, stderr)
	}()
	var termWidth, termHeight int
	fd := os.Stdin.Fd()

	if term.IsTerminal(fd) {
		oldState, err := term.MakeRaw(fd)
		if err != nil {
			return err
		}

		defer func() {
			_ = term.RestoreTerminal(fd, oldState)
		}()

		winsize, err := term.GetWinsize(fd)
		if err != nil {
			termWidth = 80
			termHeight = 24
		} else {
			termWidth = int(winsize.Width)
			termHeight = int(winsize.Height)
		}
	}

	if err = session.RequestPty("xterm", termHeight, termWidth, ssh.TerminalModes{
		ssh.ECHO: 1,
	}); err != nil {
		return err
	}

	if err = session.Shell(); err != nil {
		return err
	}

	err = session.Wait()
	//exit <- true
	if err != nil {
		return err
	}
	return nil
}

func (g *GCPClient) pollOperationStatus(operationName string) error {
	for i := 0; i < timeout; i++ {
		operation, err := g.compute.GlobalOperations.Get(g.projectName, operationName).Do()
		if err != nil {
			return fmt.Errorf("error fetching operation status: %v", err)
		}
		if operation.Error != nil {
			return fmt.Errorf("error running operation: %v", operation.Error)
		}
		if operation.Status == "DONE" {
			return nil
		}
		time.Sleep(pollingInterval)
	}
	return fmt.Errorf("timeout waiting for operation to finish")

}
func (g *GCPClient) pollZoneOperationStatus(operationName, zone string) error {
	for i := 0; i < timeout; i++ {
		operation, err := g.compute.ZoneOperations.Get(g.projectName, zone, operationName).Do()
		if err != nil {
			return fmt.Errorf("error fetching operation status: %v", err)
		}
		if operation.Error != nil {
			return fmt.Errorf("error running operation: %v", operation.Error)
		}
		if operation.Status == "DONE" {
			return nil
		}
		time.Sleep(pollingInterval)
	}
	return fmt.Errorf("timeout waiting for operation to finish")
}

// GetInstanceNatIP returns NatIP of an instance
func (g GCPClient) GetInstanceNatIP(instance, zone string) (string, error) {
	log.Debugf("Getting NatIP for instance %s", instance)
	for i := 0; i < timeout; i++ {
		res, err := g.compute.Instances.Get(g.projectName, zone, instance).Do()
		if err != nil {
			log.Errorf("GetInstanceNatIP: %s", err)
			if gcpError, ok := err.(*googleapi.Error); ok {
				if gcpError.Code == 400 {
					// Instance may not be ready yet...
					log.Debug("waiting for instance")
					time.Sleep(pollingInterval)
					continue
				}
				if gcpError.Code == 503 {
					// Timeout received when the instance has terminated
					log.Fatal("Error 503")
				}
			} else {
				time.Sleep(pollingInterval)
				continue
			}
			return "", err
		}
		for _, networkInterface := range res.NetworkInterfaces {
			for _, accessConfig := range networkInterface.AccessConfigs {
				if accessConfig.NatIP != "" {
					return accessConfig.NatIP, nil
				}
			}
		}
	}
	return "", fmt.Errorf("not found NatIP for %s", instance)
}

// SetFirewallAllowRule runs
// gcloud compute firewall-rules create ruleName --allow all --source-ranges=sourceRanges --priority=priority
func (g GCPClient) SetFirewallAllowRule(ruleName string, priority int64, sourceRanges []string) error {
	log.Infof("setting firewall %s for %s", ruleName, sourceRanges)
	for i := 0; i < timeout; i++ {
		firewall := &compute.Firewall{
			Name:         ruleName,
			SourceRanges: sourceRanges,
			Allowed: []*compute.FirewallAllowed{
				{
					IPProtocol: "icmp",
				}, {
					IPProtocol: "tcp",
				}, {
					IPProtocol: "udp",
				},
			},
			Priority: priority,
		}
		operation, err := g.compute.Firewalls.Insert(g.projectName, firewall).Do()
		if err != nil {
			if err.(*googleapi.Error).Code == 400 {
				// Firewall may not be ready yet...
				time.Sleep(pollingInterval)
				continue
			}
			if err.(*googleapi.Error).Code == 503 {
				// Timeout received when waiting for rule modification
				break
			}
			if strings.Contains(err.Error(), "alreadyExists") {
				return nil
			}
			return err
		}
		if operation.Error != nil {
			return fmt.Errorf("error running operation: %v", operation.Error)
		}
		if operation.Status == "DONE" {
			return nil
		}
	}
	return fmt.Errorf("timeout waiting for operation to finish")
}

// DeleteFirewallAllowRule runs gcloud compute firewall-rules delete ruleName
func (g GCPClient) DeleteFirewallAllowRule(ruleName string) error {
	log.Infof("deleting firewall %s", ruleName)
	for i := 0; i < timeout; i++ {
		operation, err := g.compute.Firewalls.Delete(g.projectName, ruleName).Do()
		if err != nil {
			if err.(*googleapi.Error).Code == 400 {
				// Firewall may not be ready yet...
				time.Sleep(pollingInterval)
				continue
			}
			if err.(*googleapi.Error).Code == 503 {
				// Timeout received when waiting for rule modification
				break
			}
			if strings.Contains(err.Error(), "notFound") {
				return nil
			}
			return err
		}
		if operation.Error != nil {
			return fmt.Errorf("error running operation: %v", operation.Error)
		}
		if operation.Status == "DONE" {
			return nil
		}
	}
	return fmt.Errorf("timeout waiting for operation to finish")
}
