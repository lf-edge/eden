package openevec

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/docker/docker/pkg/namesgenerator"
	"github.com/dustin/go-humanize"
	"github.com/lf-edge/eden/pkg/eve"
	"github.com/lf-edge/eden/pkg/expect"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/config"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func VolumeLs() error {
	changer := &adamChanger{}
	ctrl, dev, err := changer.getControllerAndDev()
	if err != nil {
		return fmt.Errorf("getControllerAndDev: %w", err)
	}
	state := eve.Init(ctrl, dev)
	if err := ctrl.MetricLastCallback(dev.GetID(), nil, state.MetricCallback()); err != nil {
		return fmt.Errorf("fail in get InfoLastCallback: %w", err)
	}
	if err := ctrl.InfoLastCallback(dev.GetID(), nil, state.InfoCallback()); err != nil {
		return fmt.Errorf("fail in get InfoLastCallback: %w", err)
	}
	if err := state.VolumeList(); err != nil {
		return err
	}
	return nil
}

func VolumeCreate(appLink, registry, diskSize, volumeName, volumeType, datastoreOverride string, sftpLoad, directLoad bool) error {
	changer := &adamChanger{}
	ctrl, dev, err := changer.getControllerAndDev()
	if err != nil {
		return fmt.Errorf("getControllerAndDev: %w", err)
	}
	var opts []expect.ExpectationOption
	diskSizeParsed, err := humanize.ParseBytes(diskSize)
	if err != nil {
		return err
	}
	// special case for blank volumes
	if appLink == "blank" {
		if diskSizeParsed == 0 {
			return fmt.Errorf("cannot create blank volume with 0 size, please provide --disk-size")
		}
		id, err := uuid.NewV4()
		if err != nil {
			return err
		}
		if volumeName == "" {
			// generate random name
			rand.Seed(time.Now().UnixNano())
			volumeName = namesgenerator.GetRandomName(0)
		}
		volume := &config.Volume{
			Uuid: id.String(),
			Origin: &config.VolumeContentOrigin{
				Type: config.VolumeContentOriginType_VCOT_BLANK,
			},
			Protocols:    nil,
			Maxsizebytes: int64(diskSizeParsed),
			DisplayName:  volumeName,
		}
		_ = ctrl.AddVolume(volume)
		dev.SetVolumeConfigs(append(dev.GetVolumes(), id.String()))
	} else {
		opts = append(opts, expect.WithDiskSize(int64(diskSizeParsed)))
		opts = append(opts, expect.WithImageFormat(volumeType))
		opts = append(opts, expect.WithSFTPLoad(sftpLoad))
		if !sftpLoad {
			opts = append(opts, expect.WithHTTPDirectLoad(directLoad))
		}
		opts = append(opts, expect.WithDatastoreOverride(datastoreOverride))
		registryToUse := registry
		switch registry {
		case "local":
			registryToUse = fmt.Sprintf("%s:%d", viper.GetString("registry.ip"), viper.GetInt("registry.port"))
		case "remote":
			registryToUse = ""
		}
		opts = append(opts, expect.WithRegistry(registryToUse))
		expectation := expect.AppExpectationFromURL(ctrl, dev, appLink, volumeName, opts...)
		volumeConfig := expectation.Volume()
		log.Infof("create volume %s with %s request sent", volumeConfig.DisplayName, appLink)
	}
	if err = changer.setControllerAndDev(ctrl, dev); err != nil {
		return fmt.Errorf("setControllerAndDev: %w", err)
	}
	return nil
}

func VolumeDelete(volumeName string) error {
	changer := &adamChanger{}
	ctrl, dev, err := changer.getControllerAndDev()
	if err != nil {
		return fmt.Errorf("getControllerAndDev: %w", err)
	}
	for id, el := range dev.GetVolumes() {
		volume, err := ctrl.GetVolume(el)
		if err != nil {
			return fmt.Errorf("no volume in cloud %s: %s", el, err)
		}
		if volume.DisplayName == volumeName {
			configs := dev.GetVolumes()
			utils.DelEleInSlice(&configs, id)
			dev.SetVolumeConfigs(configs)
			if err = changer.setControllerAndDev(ctrl, dev); err != nil {
				return fmt.Errorf("setControllerAndDev: %w", err)
			}
			log.Infof("volume %s delete done", volumeName)
			return nil
		}
	}
	log.Infof("not found volume with name %s", volumeName)
	return nil
}

func VolumeDetach(volumeName string) error {
	changer := &adamChanger{}
	ctrl, dev, err := changer.getControllerAndDev()
	if err != nil {
		return fmt.Errorf("getControllerAndDev: %w", err)
	}
	for _, el := range dev.GetVolumes() {
		volume, err := ctrl.GetVolume(el)
		if err != nil {
			return fmt.Errorf("no volume in cloud %s: %s", el, err)
		}
		if volume.DisplayName == volumeName {
			for _, appID := range dev.GetApplicationInstances() {
				app, err := ctrl.GetApplicationInstanceConfig(appID)
				if err != nil {
					return fmt.Errorf("no app in cloud %s: %s", el, err)
				}
				volumeRefs := app.GetVolumeRefList()
				utils.DelEleInSliceByFunction(&volumeRefs, func(i interface{}) bool {
					vol := i.(*config.VolumeRef)
					if vol.Uuid == volume.Uuid {
						purgeCounter := uint32(1)
						if app.Purge != nil {
							purgeCounter = app.Purge.Counter + 1
						}
						app.Purge = &config.InstanceOpsCmd{Counter: purgeCounter}
						log.Infof("Volume detached from %s, app will be purged", app.Displayname)
						return true
					}
					return false
				})
				app.VolumeRefList = volumeRefs
			}
			if err = changer.setControllerAndDev(ctrl, dev); err != nil {
				return fmt.Errorf("setControllerAndDev: %w", err)
			}
			return nil
		}
	}
	log.Infof("not found volume with name %s", volumeName)
	return nil
}

func VolumeAttach(appName, volumeName, mountPoint string) error {
	changer := &adamChanger{}
	ctrl, dev, err := changer.getControllerAndDev()
	if err != nil {
		return fmt.Errorf("getControllerAndDev: %w", err)
	}
	for _, el := range dev.GetVolumes() {
		volume, err := ctrl.GetVolume(el)
		if err != nil {
			return fmt.Errorf("no volume in cloud %s: %s", el, err)
		}
		if volume.DisplayName == volumeName {
			for _, appID := range dev.GetApplicationInstances() {
				app, err := ctrl.GetApplicationInstanceConfig(appID)
				if err != nil {
					return fmt.Errorf("no app in cloud %s: %s", el, err)
				}
				if app.Displayname == appName {
					purgeCounter := uint32(1)
					if app.Purge != nil {
						purgeCounter = app.Purge.Counter + 1
					}
					app.Purge = &config.InstanceOpsCmd{Counter: purgeCounter}
					app.VolumeRefList = append(app.VolumeRefList, &config.VolumeRef{Uuid: volume.Uuid, MountDir: mountPoint})
					log.Infof("Volume %s attached to %s, app will be purged", volumeName, app.Displayname)
					if err = changer.setControllerAndDev(ctrl, dev); err != nil {
						return fmt.Errorf("setControllerAndDev: %w", err)
					}
					return nil
				}
			}
			log.Infof("not found app with name %s", appName)
		}
	}
	log.Infof("not found volume with name %s", volumeName)
	return nil
}
