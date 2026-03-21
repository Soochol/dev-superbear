package mcp

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dev-superbear/nexus-backend/internal/dsl"
)

type Server struct {
	executor *dsl.Executor
}

func NewServer(executor *dsl.Executor) *Server {
	return &Server{executor: executor}
}

func (s *Server) HandleGetDSLGrammar() string {
	return s.executor.GrammarText()
}

func (s *Server) HandleListAvailableFields() string {
	fields := s.executor.AvailableFields()
	ops := s.executor.AllowedOps()

	lines := make([]string, 0, len(fields)+3)
	lines = append(lines, "Available fields:")

	// Find max name length for alignment
	maxLen := 0
	for _, f := range fields {
		if len(f.Name) > maxLen {
			maxLen = len(f.Name)
		}
	}

	for _, f := range fields {
		padding := strings.Repeat(" ", maxLen-len(f.Name))
		lines = append(lines, fmt.Sprintf("  %s%s — %s (%s)", f.Name, padding, f.Description, f.Unit))
	}

	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("All fields support operators: %s", strings.Join(ops, ", ")))
	lines = append(lines, "All fields can be used in sort by clause.")
	return strings.Join(lines, "\n")
}

func (s *Server) HandleValidateDSL(dslCode string) (string, error) {
	result := s.executor.Validate(dslCode)
	if !result.Valid {
		return "", fmt.Errorf("%s", result.Error)
	}
	b, _ := json.Marshal(result)
	return string(b), nil
}
