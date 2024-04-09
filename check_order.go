package tinkoff

import (
	"fmt"
	"io"
	"log"
)

type CheckOrderRequest struct {
	BaseRequest
	OrderID string `json:"OrderId,omitempty"`
}

func (i *CheckOrderRequest) GetValuesForToken() map[string]string {
	v := map[string]string{
		"OrderId": i.OrderID,
	}
	return v
}

type CheckOrderPaymentsResponse struct {
	PaymentID string `json:"PaymentId,omitempty"`
	Amount    int    `json:"Amount,omitempty"`
	Status    string `json:"Status,omitempty"`
}

type CheckOrderResponse struct {
	BaseResponse
	Message  string                       `json:"Message"`
	Details  string                       `json:"Details"`
	Payments []CheckOrderPaymentsResponse `json:"Payments"`
}

func (c *Client) CheckOrder(request *CheckOrderRequest) (*CheckOrderResponse, error) {
	response, err := c.PostRequest("/CheckOrder", request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	b, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(string(b))

	var res CheckOrderResponse
	err = c.decodeResponse(response, &res)
	if err != nil {
		return nil, err
	}

	err = res.Error()
	return &res, err
}
