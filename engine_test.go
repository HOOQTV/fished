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
	e.Facts["account_partner"] = "singtel"
	e.Facts["account_region"] = "SG"
	res := e.Run()

	assert.Equal(t, true, res, "should be true")
}
