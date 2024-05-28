package rolgo

type BasicDevice struct {
	Id				string				`json:"Id,omitempty"`
	Model			string				`json:"model,omitempty"`
	Manufacturer	string				`json:"manufacturer,omitempty"`
}

type Device struct {
	BasicDevice
	PowerState		string				`json:"powerState"`
	MachineState	string				`json:"machineState"`
}

