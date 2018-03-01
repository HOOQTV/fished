package fished

import (
	"io/ioutil"
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {
	raw, _ := ioutil.ReadFile("./test/tc1.json")

	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	var ruleRaw RuleRaw
	json.Unmarshal(raw, &ruleRaw)

	e := New(10)
	e.Rules = ruleRaw.Data
	e.Facts["account_partner"] = "hello"
	e.Facts["account_region"] = "ID"
	res, errs := e.Run()

	assert.Equal(t, true, res, "should be true")
	assert.Equal(t, 0, len(errs), "no errors")
}

func TestRunSpecifyEndResult(t *testing.T) {
	raw, _ := ioutil.ReadFile("./test/tc3.json")

	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	var ruleRaw RuleRaw
	json.Unmarshal(raw, &ruleRaw)

	e := New(10)
	e.Rules = ruleRaw.Data
	e.Facts["account_partner"] = "hello"
	e.Facts["account_region"] = "ID"
	res, errs := e.Run("isEligible")

	assert.Equal(t, true, res, "should be true")
	assert.Equal(t, 0, len(errs), "no errors")
}

func TestRunInvalidRule(t *testing.T) {
	raw, _ := ioutil.ReadFile("./test/tc2.json")

	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	var ruleRaw RuleRaw
	json.Unmarshal(raw, &ruleRaw)

	e := New(10)
	e.Rules = ruleRaw.Data
	e.Facts["account_partner"] = "hello"
	res, errs := e.Run()

	assert.Equal(t, nil, res, "should be true")
	assert.Equal(t, 1, len(errs), "no errors")
}
