package service

import (
	"context"
	"fmt"
)

type NLToDSLResult struct {
	DSL         string  `json:"dsl"`
	Explanation string  `json:"explanation"`
	Confidence  float64 `json:"confidence"`
}

type NLToDSLService struct{}

func NewNLToDSLService() *NLToDSLService {
	return &NLToDSLService{}
}

func (s *NLToDSLService) Convert(ctx context.Context, nlQuery string) (*NLToDSLResult, error) {
	if nlQuery == "" {
		return nil, fmt.Errorf("empty NL query")
	}
	// TODO: Implement Google ADK call
	return &NLToDSLResult{
		DSL:         "scan where volume > 1000000",
		Explanation: fmt.Sprintf(`"%s" 조건을 DSL로 변환했습니다.`, nlQuery),
		Confidence:  0.0,
	}, nil
}
