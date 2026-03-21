package mcp

import (
	"github.com/dev-superbear/nexus-backend/internal/dsl"
	"github.com/dev-superbear/nexus-backend/internal/llm/tools"
)

type Server struct {
	executor *dsl.Executor
	toolExec *tools.Executor
}

func NewServer(executor *dsl.Executor) *Server {
	return &Server{
		executor: executor,
		toolExec: tools.NewExecutor(executor),
	}
}

func (s *Server) HandleGetDSLGrammar() string {
	return s.toolExec.GetGrammar()
}

func (s *Server) HandleListAvailableFields() string {
	return s.toolExec.ListFields()
}

func (s *Server) HandleValidateDSL(dslCode string) (string, error) {
	return s.toolExec.ValidateDSL(dslCode)
}
