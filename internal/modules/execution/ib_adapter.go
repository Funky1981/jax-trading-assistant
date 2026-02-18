package execution

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// IBClient wraps HTTP calls to the IB Bridge service
type IBClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewIBClient creates a new IB Bridge HTTP client
func NewIBClient(baseURL string) *IBClient {
	return &IBClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// GetAccount retrieves account information from IB Bridge
func (c *IBClient) GetAccount(ctx context.Context) (*BrokerAccountInfo, error) {
	url := fmt.Sprintf("%s/api/v1/account", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("account request failed with status %d: %s", resp.StatusCode, body)
	}

	var result struct {
		Success bool `json:"success"`
		Account struct {
			NetLiquidation float64 `json:"net_liquidation"`
			BuyingPower    float64 `json:"buying_power"`
			Currency       string  `json:"currency"`
		} `json:"account"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode account response: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("account request unsuccessful")
	}

	return &BrokerAccountInfo{
		NetLiquidation: result.Account.NetLiquidation,
		BuyingPower:    result.Account.BuyingPower,
		Currency:       result.Account.Currency,
	}, nil
}

// PlaceOrder places an order via IB Bridge
func (c *IBClient) PlaceOrder(ctx context.Context, order *BrokerOrderRequest) (*BrokerOrderResponse, error) {
	url := fmt.Sprintf("%s/api/v1/orders", c.baseURL)

	requestBody := map[string]interface{}{
		"symbol":     order.Symbol,
		"action":     order.Action,
		"quantity":   order.Quantity,
		"order_type": order.OrderType,
	}

	if order.LimitPrice != nil {
		requestBody["limit_price"] = *order.LimitPrice
	}
	if order.StopPrice != nil {
		requestBody["stop_price"] = *order.StopPrice
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal order request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to place order: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Success bool   `json:"success"`
		OrderID int    `json:"order_id"`
		Message string `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode order response: %w", err)
	}

	return &BrokerOrderResponse{
		OrderID: result.OrderID,
		Success: result.Success,
		Message: result.Message,
	}, nil
}

// GetOrderStatus retrieves order status from IB Bridge
func (c *IBClient) GetOrderStatus(ctx context.Context, orderID int) (*BrokerOrderStatus, error) {
	url := fmt.Sprintf("%s/api/v1/orders/%d/status", c.baseURL, orderID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get order status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("order status request failed with status %d: %s", resp.StatusCode, body)
	}

	var result struct {
		Success bool `json:"success"`
		Status  struct {
			OrderID      int     `json:"order_id"`
			Status       string  `json:"status"`
			FilledQty    int     `json:"filled_qty"`
			AvgFillPrice float64 `json:"avg_fill_price"`
		} `json:"status"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode order status response: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("order status request unsuccessful")
	}

	return &BrokerOrderStatus{
		OrderID:      result.Status.OrderID,
		Status:       result.Status.Status,
		FilledQty:    result.Status.FilledQty,
		AvgFillPrice: result.Status.AvgFillPrice,
	}, nil
}

// GetPositions retrieves positions from IB Bridge
func (c *IBClient) GetPositions(ctx context.Context) (*BrokerPositionsResponse, error) {
	url := fmt.Sprintf("%s/api/v1/positions", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get positions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("positions request failed with status %d: %s", resp.StatusCode, body)
	}

	var result struct {
		Success   bool `json:"success"`
		Positions []struct {
			Symbol   string `json:"symbol"`
			Quantity int    `json:"quantity"`
		} `json:"positions"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode positions response: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("positions request unsuccessful")
	}

	positions := make([]BrokerPosition, 0, len(result.Positions))
	for _, pos := range result.Positions {
		positions = append(positions, BrokerPosition{
			Symbol:   pos.Symbol,
			Quantity: pos.Quantity,
		})
	}

	return &BrokerPositionsResponse{
		Positions: positions,
	}, nil
}
