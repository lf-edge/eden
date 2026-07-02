package cmd

import (
	"github.com/lf-edge/eden/pkg/openevec"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func newControllerCmd(configName, verbosity *string) *cobra.Command {
	cfg := &openevec.EdenSetupArgs{}
	var controllerMode string

	var controllerCmd = &cobra.Command{
		Use:               "controller",
		Short:             "interact with controller",
		Long:              `Interact with controller.`,
		PersistentPreRunE: preRunViperLoadFunction(cfg, configName, verbosity),
	}

	var edgeNode = &cobra.Command{
		Use:   "edge-node",
		Short: "manage EVE instance",
		Long:  `Manage EVE instance.`,
	}

	groups := CommandGroups{
		{
			Message: "Basic Commands",
			Commands: []*cobra.Command{
				newEdgeNodeReboot(controllerMode),
				newEdgeNodeShutdown(controllerMode),
				newEdgeNodeEVEImageUpdate(controllerMode),
				newEdgeNodeEVEImageRemove(controllerMode),
				newEdgeNodeEVEImageUpdateRetry(controllerMode),
				newEdgeNodeUpdate(controllerMode),
				newEdgeNodeGetConfig(controllerMode),
				newEdgeNodeSetConfig(),
				newEdgeNodeClusterSet(controllerMode),
				newEdgeNodeClusterClear(controllerMode),
				newEdgeNodeContentTreeAdd(controllerMode),
				newEdgeNodeAddWireless(controllerMode),
				newEdgeNodeGetOptions(controllerMode),
				newEdgeNodeSetOptions(controllerMode),
			},
		},
	}

	groups.AddTo(edgeNode)

	controllerCmd.AddCommand(edgeNode)

	controllerCmd.AddCommand(newControllerGetOptions())
	controllerCmd.AddCommand(newControllerSetOptions())

	controllerCmd.PersistentFlags().StringVarP(&controllerMode, "mode", "m", "", "mode to use [file|proto|adam|zedcloud]://<URL> (default is adam)")

	return controllerCmd
}

func newEdgeNodeReboot(controllerMode string) *cobra.Command {
	var edgeNodeReboot = &cobra.Command{
		Use:   "reboot",
		Short: "reboot EVE instance",
		Long:  `reboot EVE instance.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := openEVEC.EdgeNodeReboot(controllerMode); err != nil {
				log.Fatal(err)
			}
		},
	}
	return edgeNodeReboot
}

func newEdgeNodeEVEImageUpdateRetry(controllerMode string) *cobra.Command {
	var edgeNodeEVEImageUpdateRetry = &cobra.Command{
		Use:   "eveimage-update-retry",
		Short: "retry update of EVE image",
		Long:  `Update EVE image retry.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := openEVEC.EdgeNodeEVEImageUpdateRetry(controllerMode); err != nil {
				log.Fatal(err)
			}
		},
	}
	return edgeNodeEVEImageUpdateRetry
}

func newEdgeNodeShutdown(controllerMode string) *cobra.Command {
	var edgeNodeShutdown = &cobra.Command{
		Use:   "shutdown",
		Short: "shutdown EVE app instances",
		Long:  `shutdown EVE app instances.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := openEVEC.EdgeNodeShutdown(controllerMode); err != nil {
				log.Fatal(err)
			}
		},
	}

	return edgeNodeShutdown
}

func newEdgeNodeEVEImageUpdate(controllerMode string) *cobra.Command {
	var baseOSVersion, registry string
	var baseOSImageActivate, baseOSVDrive bool

	var edgeNodeEVEImageUpdate = &cobra.Command{
		Use:   "eveimage-update <image file or url (oci:// or file:// or http(s)://)>",
		Short: "update EVE image",
		Long:  `Update EVE image.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			baseOSImage := args[0]
			if err := openEVEC.EdgeNodeEVEImageUpdate(baseOSImage, baseOSVersion, registry, controllerMode, baseOSImageActivate, baseOSVDrive); err != nil {
				log.Fatal(err)
			}
		},
	}

	edgeNodeEVEImageUpdate.Flags().StringVarP(&baseOSVersion, "os-version", "", "", "version of ROOTFS")
	edgeNodeEVEImageUpdate.Flags().StringVar(&registry, "registry", "remote", "Select registry to use for containers (remote/local)")
	edgeNodeEVEImageUpdate.Flags().BoolVarP(&baseOSImageActivate, "activate", "", true, "activate image")
	edgeNodeEVEImageUpdate.Flags().BoolVar(&baseOSVDrive, "drive", true, "provide drive to baseOS")

	return edgeNodeEVEImageUpdate
}

func newEdgeNodeEVEImageRemove(controllerMode string) *cobra.Command {
	var baseOSVersion string

	var edgeNodeEVEImageRemove = &cobra.Command{
		Use:   "eveimage-remove <image file or url (oci:// or file:// or http(s)://)>",
		Short: "remove EVE image",
		Long:  `Remove EVE image.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			baseOSImage := args[0]
			if err := openEVEC.EdgeNodeEVEImageRemove(controllerMode, baseOSVersion, baseOSImage); err != nil {
				log.Fatal(err)
			}
		},
	}

	edgeNodeEVEImageRemove.Flags().StringVarP(&baseOSVersion, "os-version", "", "", "version of ROOTFS")

	return edgeNodeEVEImageRemove
}

func newEdgeNodeUpdate(controllerMode string) *cobra.Command {
	var deviceItems, configItems map[string]string

	var edgeNodeUpdate = &cobra.Command{
		Use:   "update --config key=value --device key=value",
		Short: "update EVE config",
		Long:  `Update EVE config.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := openEVEC.EdgeNodeUpdate(controllerMode, deviceItems, configItems); err != nil {
				log.Fatal(err)
			}
		},
	}

	configUsage := `set of key=value items.
Supported keys are defined in https://github.com/lf-edge/eve/blob/master/docs/CONFIG-PROPERTIES.md`
	deviceUsage := `set of key=value items.
Supported keys: global_profile,local_profile_server,profile_server_token`
	edgeNodeUpdate.Flags().StringToStringVar(&configItems, "config", make(map[string]string), configUsage)
	edgeNodeUpdate.Flags().StringToStringVar(&deviceItems, "device", make(map[string]string), deviceUsage)

	return edgeNodeUpdate
}

func newEdgeNodeGetOptions(controllerMode string) *cobra.Command {
	var fileWithConfig string

	var edgeNodeGetOptions = &cobra.Command{
		Use:   "get-options",
		Short: "fetch EVE options",
		Long:  `Fetch EVE options.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := openEVEC.EdgeNodeGetOptions(controllerMode, fileWithConfig); err != nil {
				log.Fatal(err)
			}
		},
	}

	edgeNodeGetOptions.Flags().StringVar(&fileWithConfig, "file", "", "save options to file")

	return edgeNodeGetOptions
}

func newEdgeNodeSetOptions(controllerMode string) *cobra.Command {
	var fileWithConfig string

	var edgeNodeSetOptions = &cobra.Command{
		Use:   "set-options",
		Short: "set EVE options",
		Long:  `Set EVE options.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := openEVEC.EdgeNodeSetOptions(controllerMode, fileWithConfig); err != nil {
				log.Fatal(err)
			}
		},
	}

	edgeNodeSetOptions.Flags().StringVar(&fileWithConfig, "file", "", "set options from file")

	return edgeNodeSetOptions
}

func newControllerGetOptions() *cobra.Command {
	var fileWithConfig string

	var controllerGetOptions = &cobra.Command{
		Use:   "get-options",
		Short: "fetch controller options",
		Long:  `Fetch controller options.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := openEVEC.ControllerGetOptions(fileWithConfig); err != nil {
				log.Fatal(err)
			}
		},
	}

	controllerGetOptions.Flags().StringVar(&fileWithConfig, "file", "", "save options to file")

	return controllerGetOptions
}

func newControllerSetOptions() *cobra.Command {
	var fileWithConfig string

	var controllerSetOptions = &cobra.Command{
		Use:   "set-options",
		Short: "set controller options",
		Long:  `Set controller options.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := openEVEC.ControllerSetOptions(fileWithConfig); err != nil {
				log.Fatal(err)
			}
		},
	}

	controllerSetOptions.Flags().StringVar(&fileWithConfig, "file", "", "set options from file")

	return controllerSetOptions
}

func newEdgeNodeGetConfig(controllerMode string) *cobra.Command {
	var fileWithConfig string

	var edgeNodeGetConfig = &cobra.Command{
		Use:   "get-config",
		Short: "fetch EVE config",
		Long:  `Fetch EVE config.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := openEVEC.EdgeNodeGetConfig(controllerMode, fileWithConfig); err != nil {
				log.Fatal(err)
			}
		},
	}

	edgeNodeGetConfig.Flags().StringVar(&fileWithConfig, "file", "", "save config to file")

	return edgeNodeGetConfig
}

func newEdgeNodeSetConfig() *cobra.Command {
	var fileWithConfig string

	var edgeNodeSetConfig = &cobra.Command{
		Use:   "set-config",
		Short: "set EVE config",
		Long:  `Set EVE config.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := openEVEC.EdgeNodeSetConfig(fileWithConfig); err != nil {
				log.Fatal(err)
			}
		},
	}

	edgeNodeSetConfig.Flags().StringVar(&fileWithConfig, "file", "", "set config from file")

	return edgeNodeSetConfig
}

func newEdgeNodeClusterSet(controllerMode string) *cobra.Command {
	var clusterType string

	var edgeNodeClusterSet = &cobra.Command{
		Use:   "cluster-set",
		Short: "set EdgeNodeCluster config on the device",
		Long: `Set EdgeNodeCluster config on the device.

This pushes a loopback-stub EdgeNodeCluster (clusterIpPrefix=127.0.0.1/32,
joinServerIp=127.0.0.1, clusterInterface=lo, stable clusterId) with the
selected --type. EVE-side, the publishing of an ENCC with Valid=true and
a non-ReplicatedStorage clusterType makes volumemgr's startup wait
short-circuit the longhorn-readiness sub-wait that otherwise costs ~10
minutes on single-node EVE-k where longhorn is not installed.

Workaround for lf-edge/eve#6018 — the cleaner long-term fix is on the
EVE side (flip volumemgr's default waitForLhFlag to false).`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := openEVEC.EdgeNodeClusterSet(controllerMode, clusterType); err != nil {
				log.Fatal(err)
			}
		},
	}

	edgeNodeClusterSet.Flags().StringVar(&clusterType, "type", "k3sbase",
		"cluster type: k3sbase | replicated-storage | ha | none")

	return edgeNodeClusterSet
}

func newEdgeNodeContentTreeAdd(controllerMode string) *cobra.Command {
	var registry, contentTreeName, datastoreOverride string
	var sftpLoad, directLoad bool

	var edgeNodeContentTreeAdd = &cobra.Command{
		Use:   "content-tree-add <(docker|http(s)|file)://(<TAG>[:<VERSION>] | <URL> | <PATH>)>",
		Short: "register a standalone ContentTree in the device config (no Volume)",
		Long: `Register a standalone ContentTree (no associated Volume or AppInstance)
in the EdgeDevConfig. Pillar downloads ContentTrees eagerly and looks up
blobs by SHA256, so a subsequent 'eden pod deploy' against the same image
URL (or any image sharing the SHA) reuses the pre-staged blobs without
re-downloading.

Use case: pre-stage a content tree on EVE-kvm, push a cross-HV upgrade to
EVE-k, and deploy an app referencing the same image on EVE-k — the
content tree survives the upgrade and is reused. Avoids the
Volume/PVC machinery entirely.`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := openEVEC.EdgeNodeContentTreeAdd(controllerMode, args[0], registry, contentTreeName, datastoreOverride, sftpLoad, directLoad); err != nil {
				log.Fatal(err)
			}
		},
	}

	edgeNodeContentTreeAdd.Flags().StringVarP(&contentTreeName, "name", "n", "", "display name of content tree (defaults to image name)")
	edgeNodeContentTreeAdd.Flags().StringVar(&registry, "registry", "remote", "Select registry to use for containers (remote/local)")
	edgeNodeContentTreeAdd.Flags().StringVar(&datastoreOverride, "datastoreOverride", "", "Override datastore path (when Eden and EVE see different URLs)")
	edgeNodeContentTreeAdd.Flags().BoolVar(&sftpLoad, "sftp", false, "force eserver to use sftp")
	edgeNodeContentTreeAdd.Flags().BoolVar(&directLoad, "direct", true, "Use direct download for image instead of eserver")
	return edgeNodeContentTreeAdd
}

func newEdgeNodeAddWireless(controllerMode string) *cobra.Command {
	var portName, ssid, username, password string
	var useEncryptCert bool

	var edgeNodeAddWireless = &cobra.Command{
		Use:   "add-wireless",
		Short: "add a WiFi device port with ENCRYPTED credentials (non-mgmt)",
		Long: `Inject a WiFi device port into the EdgeDevConfig with its credentials
encrypted (ECDH) against the device's certificate: a wireless PhysicalIO, a
NetworkConfig whose WifiConfig carries the encrypted cipherData, and a
non-management SystemAdapter. EVE decrypts the credentials at device-config
ingest using /persist/certs/ecdh.*, so this exercises credential decryption
independent of any app and of physical radio presence (used by the kvm-to-k
F9 persist-restore test to prove decryption from restored files).`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := openEVEC.AddWirelessPort(controllerMode, portName, ssid, username, password, useEncryptCert); err != nil {
				log.Fatal(err)
			}
		},
	}

	edgeNodeAddWireless.Flags().StringVar(&portName, "port", "wlan0", "phy/logical label and interface name of the wireless port")
	edgeNodeAddWireless.Flags().StringVar(&ssid, "ssid", "eden-test-ssid", "WiFi SSID")
	edgeNodeAddWireless.Flags().StringVar(&username, "username", "", "EAP identity/username (encrypted)")
	edgeNodeAddWireless.Flags().StringVar(&password, "password", "eden-test-psk", "WiFi PSK/password (encrypted)")
	edgeNodeAddWireless.Flags().BoolVar(&useEncryptCert, "use-encrypt-cert", true, "encrypt against the controller encrypt cert (CONTROLLER_ECDH_EXCHANGE)")
	return edgeNodeAddWireless
}

func newEdgeNodeClusterClear(controllerMode string) *cobra.Command {
	var edgeNodeClusterClear = &cobra.Command{
		Use:   "cluster-clear",
		Short: "clear EdgeNodeCluster config on the device",
		Long:  `Clear EdgeNodeCluster config on the device.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := openEVEC.EdgeNodeClusterClear(controllerMode); err != nil {
				log.Fatal(err)
			}
		},
	}

	return edgeNodeClusterClear
}
