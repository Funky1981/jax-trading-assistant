package utcp

type BrokerService struct {
	client Client
}

func NewBrokerService(c Client) *BrokerService {
	return &BrokerService{client: c}
}
