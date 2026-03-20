package llm

import _ "embed"

//go:embed prompts/nl-to-dsl.txt
var NLToDSLPrompt string

//go:embed prompts/explain.txt
var ExplainPrompt string
