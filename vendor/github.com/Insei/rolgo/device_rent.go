package rolgo

const deviceRentsBasePath = "devices/rents/"
const projectIdHeader = "X-Project-Id"

type DeviceRent struct {
	Id           string `json:"id"`
	Name         string `json:"name"`
	ProjectId    string `json:"projectId"`
	PowerState   string `json:"powerState"`
	MachineState string `json:"machineState"`
}

type ConsoleLog struct {
	DateUtc string
	Text    string
}

type DeviceRentsResponse struct {
	ApiResponse
	Data []DeviceRent `json:"data"`
}

type DeviceRentResponse struct {
	ApiResponse
	Data *DeviceRent `json:"data"`
}

type ConsoleOutputResponse struct {
	ApiResponse
	Data []ConsoleLog `json:"data"`
}

type DeviceRentUpdateRequest struct {
	Name string `json:"name,omitempty"`
}

type DeviceRentCreateRequest struct {
	Name         string `json:"name"`
	Model        string `json:"model"`
	Manufacturer string `json:"manufacturer"`
	IpxeUrl      string `json:"ipxeUrl"`
}

type RentsService interface {
	List(string) ([]DeviceRent, error)
	Get(string, string) (*DeviceRent, error)
	Create(string, *DeviceRentCreateRequest) (*DeviceRent, error)
	Update(string, string, *DeviceRentUpdateRequest) (*DeviceRent, error)
	Release(string, string) error
	GetConsoleOutput(string, string) ([]string, error)
}

type RentsServiceOp struct {
	client *Client
}

func (s *RentsServiceOp) Get(projectId string, rentId string) (*DeviceRent, error) {
	resty := s.client.resty
	apiResp := new(DeviceRentResponse)
	_, err := resty.R().
		SetHeader(projectIdHeader, projectId).
		SetResult(apiResp).
		Get(resty.BaseURL + deviceRentsBasePath + rentId)

	if err != nil {
		return nil, err
	}

	return apiResp.Data, nil
}

func (s *RentsServiceOp) Create(projectId string, request *DeviceRentCreateRequest) (*DeviceRent, error) {
	resty := s.client.resty
	apiResp := new(DeviceRentResponse)
	_, err := resty.R().
		SetHeader(projectIdHeader, projectId).
		SetBody(request).
		SetResult(apiResp).
		Post(resty.BaseURL + deviceRentsBasePath)

	if err != nil {
		return nil, err
	}

	return apiResp.Data, nil
}

func (s *RentsServiceOp) Update(projectId string, rentId string, request *DeviceRentUpdateRequest) (*DeviceRent, error) {
	resty := s.client.resty
	apiResp := new(DeviceRentResponse)
	_, err := resty.R().
		SetHeader(projectIdHeader, projectId).
		SetBody(request).
		SetResult(apiResp).
		Put(resty.BaseURL + deviceRentsBasePath + rentId)

	if err != nil {
		return nil, err
	}

	return apiResp.Data, nil
}

func (s *RentsServiceOp) Release(projectId string, rentId string) error {
	resty := s.client.resty
	apiResp := new(ApiResponse)
	_, err := resty.R().
		SetHeader(projectIdHeader, projectId).
		SetResult(apiResp).
		Delete(resty.BaseURL + deviceRentsBasePath + rentId)

	return err
}

func (s *RentsServiceOp) List(projectId string) ([]DeviceRent, error) {
	resty := s.client.resty
	apiResp := new(DeviceRentsResponse)
	_, err := resty.R().
		SetHeader(projectIdHeader, projectId).
		SetResult(apiResp).
		Get(resty.BaseURL + deviceRentsBasePath)

	if err != nil {
		return nil, err
	}

	return apiResp.Data, nil
}

func (s *RentsServiceOp) GetConsoleOutput(projectId string, rentId string) ([]string, error) {
	resty := s.client.resty
	apiResp := new(ConsoleOutputResponse)
	_, err := resty.R().
		SetHeader(projectIdHeader, projectId).
		SetResult(apiResp).
		Get(resty.BaseURL + deviceRentsBasePath + rentId + "/console/output")
	if err != nil {
		return nil, err
	}
	consoleOutput := []string{}
	for _, consoleLine := range apiResp.Data {
		consoleOutput = append(consoleOutput, consoleLine.Text)
	}
	return consoleOutput, err
}
