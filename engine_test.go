package fished

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewEngine(t *testing.T) {
	e, err := New()
	assert.NoError(t, err)
	assert.NotEmpty(t, e)
}

func TestSetFacts(t *testing.T) {
	e, err := New()
	assert.NoError(t, err)

	facts := map[string]interface{}{"hello": "world!"}
	err = e.SetFacts(facts)
	assert.NoError(t, err)
}

func TestSetRule(t *testing.T) {
	e, err := New()
	assert.NoError(t, err)

	Rules := make([]Rule, 5)
	Rules = append(Rules, Rule{Input: []string{"Hello", "World"}, Expression: "Hello == World", Output: map[string]RuleOutput{"Output": RuleOutput{Value: true}}})

	err = e.SetRules(Rules)
	assert.NoError(t, err)
}

func TestRun(t *testing.T) {
	e, err := New()
	assert.NoError(t, err)

	Rules := make([]Rule, 5)
	Rules = append(Rules, Rule{Input: []string{"Hello", "World"}, Expression: "Hello == World", Output: map[string]RuleOutput{"Output": RuleOutput{Value: true}}})

	err = e.SetRules(Rules)
	assert.NoError(t, err)

	facts := map[string]interface{}{"Hello": "world!", "World": "world!"}
	err = e.SetFacts(facts)
	assert.NoError(t, err)

	err = e.Run()
	assert.NoError(t, err)
}
