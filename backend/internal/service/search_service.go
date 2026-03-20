package service

import (
	"context"

	"github.com/dev-superbear/nexus-backend/internal/dsl"
)

type SearchService struct {
	executor *dsl.Executor
}

func NewSearchService(executor *dsl.Executor) *SearchService {
	if executor == nil {
		executor = dsl.NewExecutor()
	}
	return &SearchService{executor: executor}
}

func (s *SearchService) Validate(ctx context.Context, dslCode string) dsl.ValidationResult {
	return s.executor.Validate(dslCode)
}

func (s *SearchService) ParseScanQuery(ctx context.Context, dslCode string) (*dsl.ParsedScanQuery, error) {
	return s.executor.ParseScan(dslCode)
}

func (s *SearchService) Execute(ctx context.Context, dslCode string) ([]dsl.SearchResult, error) {
	return s.executor.Execute(ctx, dslCode)
}
