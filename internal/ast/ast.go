package ast

type RedirectionType int

const (
	RedirIn RedirectionType = iota
	RedirOut
	RedirAppend
	RedirErr
	RedirErrApp
	RedirErrToOut
	RedirAll
)

type Redirection struct {
	Type   RedirectionType
	Target string
}

type Command struct {
	Args       []string
	Env        map[string]string
	Redirects  []Redirection
	Background bool
}

type Pipeline struct {
	Commands   []*Command
	Background bool
}

type Operator int

const (
	OpNone Operator = iota
	OpSemi
	OpAnd
	OpOr
)

type ChainedPipeline struct {
	Pipeline *Pipeline
	Operator Operator
}

type CommandList struct {
	Pipelines []ChainedPipeline
}
