package dexter

import (
	"context"
	"errors"
)

// MockClient implements a mock Dexter client for testing
type MockClient struct {
	ResearchCompanyFunc  func(ctx context.Context, input ResearchCompanyInput) (ResearchCompanyOutput, error)
	CompareCompaniesFunc func(ctx context.Context, input CompareCompaniesInput) (CompareCompaniesOutput, error)
	HealthFunc           func(ctx context.Context) error
}

func (m *MockClient) ResearchCompany(ctx context.Context, input ResearchCompanyInput) (ResearchCompanyOutput, error) {
	if m.ResearchCompanyFunc != nil {
		return m.ResearchCompanyFunc(ctx, input)
	}
	return ResearchCompanyOutput{}, errors.New("mock research not implemented")
}

func (m *MockClient) CompareCompanies(ctx context.Context, input CompareCompaniesInput) (CompareCompaniesOutput, error) {
	if m.CompareCompaniesFunc != nil {
		return m.CompareCompaniesFunc(ctx, input)
	}
	return CompareCompaniesOutput{}, errors.New("mock compare not implemented")
}

func (m *MockClient) Health(ctx context.Context) error {
	if m.HealthFunc != nil {
		return m.HealthFunc(ctx)
	}
	return nil
}
