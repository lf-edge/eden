package projects

import "github.com/lf-edge/eden/pkg/device"

//EdgeNodeDescription must be defined in config file
type EdgeNodeDescription struct {
	Key    string
	Serial string
	Model  string
}

//GetEdgeNode returns EdgeNode for provided EdgeNodeDescription based on onboarding key (if exists) or name
func (nodeDescription *EdgeNodeDescription) GetEdgeNode(tc *TestContext) *device.Ctx {
	ctrl := tc.GetController()
	if nodeDescription.Key != "" {
		id, err := ctrl.DeviceGetByOnboard(nodeDescription.Key)
		if err != nil {
			return nil
		}
		dev, err := ctrl.GetDeviceUUID(id)
		if err != nil {
			return nil
		}
		return dev
	}
	return nil
}

//EdgeNodeOption is type to use for creation of device.Ctx
type EdgeNodeOption func(description *device.Ctx)

//WithNodeDescription sets device info
func (tc *TestContext) WithNodeDescription(nodeDescription *EdgeNodeDescription) EdgeNodeOption {
	return func(d *device.Ctx) {
		d.SetDevModel(nodeDescription.Model)
		d.SetOnboardKey(nodeDescription.Key)
		d.SetSerial(nodeDescription.Serial)
	}
}

//WithCurrentProject sets project info
func (tc *TestContext) WithCurrentProject() EdgeNodeOption {
	return func(d *device.Ctx) {
		d.SetProject(tc.project.name)
	}
}

//WithDeviceModel sets device model info
func (tc *TestContext) WithDeviceModel(devModel string) EdgeNodeOption {
	return func(d *device.Ctx) {
		d.SetDevModel(devModel)
	}
}
