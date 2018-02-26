package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/knetic/govaluate"

	jsoniter "github.com/json-iterator/go"
)

type RuleRaw struct {
	Data []Rule `json:"data"`
}

func main() {
	raw, err := ioutil.ReadFile("./rule.json")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	var rules RuleRaw
	json.Unmarshal(raw, &rules)

	startTime := time.Now()
	e := New(4) // limit of rules that can be run simultan
	e.Rules = rules.Data
	e.Facts = make(map[string]interface{})
	e.RuleFunctions = make(map[string]govaluate.ExpressionFunction)
	e.Facts["account_partner"] = "telkom"
	e.Facts["account_region"] = "SG"
	fmt.Println("Result:", e.Run())
	endTime := time.Now()
	diff := endTime.Sub(startTime)
	fmt.Println("total time taken ", diff.Seconds()*1000, "miliseconds")
}
