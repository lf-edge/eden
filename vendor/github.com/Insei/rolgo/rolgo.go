package rolgo

import (
	"errors"
	"github.com/go-resty/resty/v2"
	"os"
)

// Client the base API Client
type Client struct {
	resty *resty.Client

	Rents RentsService
}

func NewClient() (*Client, error) {
	var c = new(Client)

	url := os.Getenv("ROL_API_URL")
	key := os.Getenv("ROL_API_KEY")
	if url == "" || key == "" {
		return nil, errors.New("it is necessary to assign ROL_API_URL and ROL_API_KEY environment variables")
	}

	c.resty = resty.New()
	c.resty.BaseURL = url
	c.resty.SetHeader("X-API-Key", key)
	c.resty.OnAfterResponse(func(c *resty.Client, resp *resty.Response) error {
		if resp.IsError() {
			return errors.New(resp.Status())
		}

		apiResp := new(ApiResponse)
		c.JSONUnmarshal(resp.Body(), apiResp)
		if apiResp.Status.Code != 0 {
			return errors.New(apiResp.Status.Message)
		}

		return nil
	})

	c.Rents = &RentsServiceOp{client: c}

	return c, nil
}
