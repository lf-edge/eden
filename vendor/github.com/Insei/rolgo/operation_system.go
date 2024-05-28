package rolgo

type OSOption struct {
	Name string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

type OS struct {
	Name int  `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
	Arch string `json:"arch,omitempty"`
	Options []OSOption `json:"options,omitempty"`
}