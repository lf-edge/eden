package projects

import "github.com/lf-edge/eden/pkg/device"

//EdgeNodeDescription must be defined in config file
type EdgeNodeDescription struct {
	Name   string
	Key    string
	Serial string
	Model  string
}

//EdgeNodeOption is type to use for creation of device.Ctx
type EdgeNodeOption func(description *device.Ctx)

func (ctx *TestContext) WithNodeDescription(nodeDescription *EdgeNodeDescription) EdgeNodeOption {
	return func(d *device.Ctx) {
		d.SetName(nodeDescription.Name)
		d.SetDevModel(nodeDescription.Model)
		d.SetOnboardKey(nodeDescription.Key)
		d.SetSerial(nodeDescription.Serial)
	}
}

func (ctx *TestContext) WithCurrentProject() EdgeNodeOption {
	return func(d *device.Ctx) {
		d.SetProject(ctx.project.name)
	}
}

func (ctx *TestContext) WithDeviceModel(devModel string) EdgeNodeOption {
	return func(d *device.Ctx) {
		d.SetDevModel(devModel)
	}
}
