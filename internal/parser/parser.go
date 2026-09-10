package parser

import (
	"bush/internal/ast"
	"fmt"
	"strings"
)

type Parser struct {
	tokens []Token
	pos    int
}

func NewParser(tokens []Token) *Parser {
	return &Parser{tokens: tokens, pos: 0}
}

func (p *Parser) current() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) advance() Token {
	tok := p.current()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return tok
}

func (p *Parser) match(types ...TokenType) bool {
	cur := p.current().Type
	for _, t := range types {
		if cur == t {
			p.advance()
			return true
		}
	}
	return false
}

func Parse(input string) (*ast.CommandList, error) {
	tokens, err := Tokenize(input)
	if err != nil {
		return nil, err
	}

	p := NewParser(tokens)
	cmdList := &ast.CommandList{}

	for p.current().Type != TokenEOF {
		pipeline, err := p.parsePipeline()
		if err != nil {
			return nil, err
		}
		if pipeline == nil || len(pipeline.Commands) == 0 {
			break
		}

		op := ast.OpNone
		if p.match(TokenAnd) {
			op = ast.OpAnd
		} else if p.match(TokenOr) {
			op = ast.OpOr
		} else if p.match(TokenSemi) {
			op = ast.OpSemi
		} else if p.match(TokenAmp) {
			pipeline.Background = true
			op = ast.OpNone
		}

		cmdList.Pipelines = append(cmdList.Pipelines, ast.ChainedPipeline{
			Pipeline: pipeline,
			Operator: op,
		})
	}

	return cmdList, nil
}

func (p *Parser) parsePipeline() (*ast.Pipeline, error) {
	pipeline := &ast.Pipeline{}

	for {
		cmd, err := p.parseCommand()
		if err != nil {
			return nil, err
		}
		if cmd != nil {
			pipeline.Commands = append(pipeline.Commands, cmd)
		}

		if p.current().Type == TokenPipe {
			p.advance()
			continue
		}
		break
	}

	return pipeline, nil
}

func (p *Parser) parseCommand() (*ast.Command, error) {
	cmd := &ast.Command{
		Env: make(map[string]string),
	}

	for p.current().Type != TokenEOF {
		tok := p.current()

		if tok.Type == TokenPipe || tok.Type == TokenAnd || tok.Type == TokenOr || tok.Type == TokenSemi || tok.Type == TokenAmp {
			break
		}

		switch tok.Type {
		case TokenRedirIn:
			p.advance()
			target := p.advance()
			if target.Type != TokenWord {
				return nil, fmt.Errorf("expected filename after '<'")
			}
			cmd.Redirects = append(cmd.Redirects, ast.Redirection{Type: ast.RedirIn, Target: target.Value})
			continue
		case TokenRedirOut:
			p.advance()
			target := p.advance()
			if target.Type != TokenWord {
				return nil, fmt.Errorf("expected filename after '>'")
			}
			cmd.Redirects = append(cmd.Redirects, ast.Redirection{Type: ast.RedirOut, Target: target.Value})
			continue
		case TokenRedirAppend:
			p.advance()
			target := p.advance()
			if target.Type != TokenWord {
				return nil, fmt.Errorf("expected filename after '>>'")
			}
			cmd.Redirects = append(cmd.Redirects, ast.Redirection{Type: ast.RedirAppend, Target: target.Value})
			continue
		case TokenRedirErr:
			p.advance()
			target := p.advance()
			if target.Type != TokenWord {
				return nil, fmt.Errorf("expected filename after '2>'")
			}
			cmd.Redirects = append(cmd.Redirects, ast.Redirection{Type: ast.RedirErr, Target: target.Value})
			continue
		case TokenRedirErrApp:
			p.advance()
			target := p.advance()
			if target.Type != TokenWord {
				return nil, fmt.Errorf("expected filename after '2>>'")
			}
			cmd.Redirects = append(cmd.Redirects, ast.Redirection{Type: ast.RedirErrApp, Target: target.Value})
			continue
		case TokenRedirErrToOut:
			p.advance()
			cmd.Redirects = append(cmd.Redirects, ast.Redirection{Type: ast.RedirErrToOut, Target: "&1"})
			continue
		case TokenRedirAll:
			p.advance()
			target := p.advance()
			if target.Type != TokenWord {
				return nil, fmt.Errorf("expected filename after '&>'")
			}
			cmd.Redirects = append(cmd.Redirects, ast.Redirection{Type: ast.RedirAll, Target: target.Value})
			continue
		}

		if tok.Type == TokenWord {
			val := p.advance().Value

			if len(cmd.Args) == 0 && strings.Contains(val, "=") && !strings.HasPrefix(val, "=") {
				parts := strings.SplitN(val, "=", 2)
				if isValidIdent(parts[0]) {
					cmd.Env[parts[0]] = parts[1]
					continue
				}
			}
			cmd.Args = append(cmd.Args, val)
			continue
		}

		break
	}

	if len(cmd.Args) == 0 && len(cmd.Redirects) == 0 && len(cmd.Env) == 0 {
		return nil, nil
	}

	return cmd, nil
}

func isValidIdent(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i, r := range s {
		if i == 0 && (r >= '0' && r <= '9') {
			return false
		}
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' {
			return false
		}
	}
	return true
}
