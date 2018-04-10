package fished

import (
	"errors"
	"io/ioutil"
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
)

var c = &Config{
	Worker:         1,
	WorkerPoolSize: 20,
	Cache:          false,
}

func TestRun(t *testing.T) {
	raw, _ := ioutil.ReadFile("./test/tc1.json")

	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	var ruleRaw RuleRaw
	json.Unmarshal(raw, &ruleRaw)

	e := New(c)
	e.SetRules(ruleRaw.Data)
	f := make(map[string]interface{})
	f["account_partner"] = "hello"
	f["account_region"] = "ID"
	e.SetFacts(f)
	res, errs := e.Run()

	assert.Equal(t, true, res, "should be true")
	assert.Equal(t, 0, len(errs), "no errors")
}

func TestRunNil(t *testing.T) {
	raw, _ := ioutil.ReadFile("./test/tc1.json")

	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	var ruleRaw RuleRaw
	json.Unmarshal(raw, &ruleRaw)

	e := New(nil)
	e.SetRules(ruleRaw.Data)
	f := make(map[string]interface{})
	f["account_partner"] = "hello"
	f["account_region"] = "ID"
	e.SetFacts(f)
	res, errs := e.Run()

	assert.Equal(t, true, res, "should be true")
	assert.Equal(t, 0, len(errs), "no errors")
}

func TestRunInvalidRule(t *testing.T) {
	raw, _ := ioutil.ReadFile("./test/tc2.json")

	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	var ruleRaw RuleRaw
	json.Unmarshal(raw, &ruleRaw)

	e := New(c)
	e.SetRules(ruleRaw.Data)
	f := make(map[string]interface{})
	f["account_partner"] = "hello"
	e.SetFacts(f)
	res, errs := e.Run()

	assert.Equal(t, nil, res, "should be nil")
	assert.Equal(t, 1, len(errs), "no errors")
}

func TestRunSpecifyEndResult(t *testing.T) {
	raw, _ := ioutil.ReadFile("./test/tc3.json")

	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	var ruleRaw RuleRaw
	json.Unmarshal(raw, &ruleRaw)

	e := New(c)
	e.SetRules(ruleRaw.Data)
	f := make(map[string]interface{})
	f["account_partner"] = "hello"
	f["account_region"] = "ID"
	e.SetFacts(f)
	res, errs := e.Run("isEligible")

	assert.Equal(t, true, res, "should be true")
	assert.Equal(t, 0, len(errs), "no errors")
}

func TestRuleFunction(t *testing.T) {
	raw, _ := ioutil.ReadFile("./test/tc4.json")

	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	var ruleRaw RuleRaw
	json.Unmarshal(raw, &ruleRaw)

	e := New(c)
	f := make(map[string]interface{})
	f["example"] = "random"
	rf := make(map[string]RuleFunction)
	rf["set"] = func(arguments ...interface{}) (interface{}, error) {
		if len(arguments) == 1 {
			return arguments[0], nil
		}
		return nil, errors.New("Lack of arguments")
	}
	e.SetRules(ruleRaw.Data)
	e.SetFacts(f)
	e.SetRuleFunction(rf)

	res, errs := e.Run()

	assert.Equal(t, true, res, "should be true")
	assert.Equal(t, 0, len(errs), "no errors")
}

func BenchmarkFullRemake(b *testing.B) {
	raw, _ := ioutil.ReadFile("./test/tc1.json")

	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	var ruleRaw RuleRaw
	json.Unmarshal(raw, &ruleRaw)

	for i := 0; i < b.N; i++ {
		e := New(c)
		e.SetRules(ruleRaw.Data)
		f := make(map[string]interface{})
		f["account_partner"] = "hello"
		f["account_region"] = "ID"
		e.SetFacts(f)
		e.Run()
	}
}

func BenchmarkResetFacts(b *testing.B) {
	raw, _ := ioutil.ReadFile("./test/tc1.json")

	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	var ruleRaw RuleRaw
	json.Unmarshal(raw, &ruleRaw)

	e := New(c)
	e.SetRules(ruleRaw.Data)
	for i := 0; i < b.N; i++ {
		f := make(map[string]interface{})
		f["account_partner"] = "hello"
		f["account_region"] = "ID"
		e.SetFacts(f)
		e.Run()
	}
}

func BenchmarkRun1(b *testing.B) {
	raw, _ := ioutil.ReadFile("./test/tc1.json")

	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	var ruleRaw RuleRaw
	json.Unmarshal(raw, &ruleRaw)

	e := New(c)
	e.SetRules(ruleRaw.Data)
	f := make(map[string]interface{})
	f["account_partner"] = "hello"
	f["account_region"] = "ID"
	e.SetFacts(f)

	for i := 0; i < b.N; i++ {
		e.Run()
	}
}

func BenchmarkRun1Cached(b *testing.B) {
	raw, _ := ioutil.ReadFile("./test/tc1.json")

	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	var ruleRaw RuleRaw
	json.Unmarshal(raw, &ruleRaw)

	c.Cache = true
	e := New(c)
	e.SetRules(ruleRaw.Data)
	f := make(map[string]interface{})
	f["account_partner"] = "hello"
	f["account_region"] = "ID"
	e.SetFacts(f)

	for i := 0; i < b.N; i++ {
		e.Run()
	}
}

func BenchmarkRun10(b *testing.B) {
	raw, _ := ioutil.ReadFile("./test/tc1.json")

	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	var ruleRaw RuleRaw
	json.Unmarshal(raw, &ruleRaw)

	c.Worker = 10
	e := New(c)
	e.SetRules(ruleRaw.Data)
	f := make(map[string]interface{})
	f["account_partner"] = "hello"
	f["account_region"] = "ID"
	e.SetFacts(f)

	for i := 0; i < b.N; i++ {
		e.Run()
	}
}

func BenchmarkRun10Cached(b *testing.B) {
	raw, _ := ioutil.ReadFile("./test/tc1.json")

	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	var ruleRaw RuleRaw
	json.Unmarshal(raw, &ruleRaw)

	c.Worker = 10
	c.Cache = true
	e := New(c)
	e.SetRules(ruleRaw.Data)
	f := make(map[string]interface{})
	f["account_partner"] = "hello"
	f["account_region"] = "ID"
	e.SetFacts(f)

	for i := 0; i < b.N; i++ {
		e.Run()
	}
}
