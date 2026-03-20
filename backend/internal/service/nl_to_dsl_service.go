package service

import (
	"context"

	"github.com/dev-superbear/nexus-backend/internal/llm"
)

type NLToDSLService struct {
	provider llm.Provider
}

func NewNLToDSLService(provider llm.Provider) *NLToDSLService {
	return &NLToDSLService{provider: provider}
}

func (s *NLToDSLService) Stream(ctx context.Context, query string) (<-chan llm.Event, error) {
	return s.provider.NLToDSL(ctx, query)
}

func (s *NLToDSLService) Explain(ctx context.Context, dsl string) (string, error) {
	return s.provider.Explain(ctx, dsl)
}
