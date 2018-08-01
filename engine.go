package fished

import (
	"encoding/hex"
	"strings"

	"github.com/hashicorp/golang-lru"
	"github.com/oleksandr/conditions"
)

type (
	// Rule ...
	Rule struct {
		ID                  string
		Input               []string
		Expression          string
		EvaluatedExpression *conditions.Parser
		Output              map[string]RuleOutput
	}

	// RuleOutput ...
	RuleOutput struct {
		Type              string
		Value             *string
		Function          *string
		Parameter         *string
		ConstantParameter *string
	}

	// ruleObject ...
	ruleObject struct {
		Input    map[string]interface{}
		RuleHash []byte
	}

	// Engine ...
	Engine struct {
		ruleFunctions map[string]RuleFunction
		workingMemory *lru.Cache
		rules         *lru.Cache
		usedRule      []string
	}

	// RuleFunction ..
	RuleFunction func(...interface{}) (interface{}, error)
)

// New ...
func New() (*Engine, error) {
	return &Engine{}, nil
}

// SetRules ...
func (e *Engine) SetRules(r []Rule) error {
	var err error

	e.rules, err = lru.New(64 * 1024)
	if err != nil {
		return err
	}

	e.usedRule = make([]string, len(r))
	for _, v := range r {
		go func() {
			ruleBytes, _ := getBytes(v)
			hash := getMD5Hash(ruleBytes)
			v.ID = hex.EncodeToString(hash)

			v.EvaluatedExpression = conditions.NewParser(strings.NewReader(v.Expression))

			e.rules.Add(v.ID, v)
		}()
	}
	return nil
}

// SetFacts ...
func (e *Engine) SetFacts(f map[string]interface{}) error {
	var err error
	e.workingMemory, err = lru.New(1024)
	if err != nil {
		return err
	}

	for k, v := range f {
		e.workingMemory.Add(k, v)
	}
	return nil
}

// SetRuleFunctions ...
func (e *Engine) SetRuleFunctions(rf map[string]RuleFunction) error {
	e.ruleFunctions = make(map[string]RuleFunction)

	for k, v := range rf {
		e.ruleFunctions[k] = v
	}

	return nil
}
