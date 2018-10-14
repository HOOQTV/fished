package fished

import (
	"io/ioutil"
	"os"
	"runtime"
	"testing"

	"github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func TestRun(t *testing.T) {
	tc := []struct {
		Name           string
		TCFile         string
		Facts          map[string]interface{}
		RuleFunction   map[string]RuleFunction
		Target         string
		Worker         int
		ExpectedResult interface{}
		IsError        bool
	}{
		{
			Name:   "test",
			TCFile: "./test/tc1.json",
			Facts: map[string]interface{}{
				"account_partner": "hello",
				"account_region":  "ID",
				"flight_type":     "free",
			},
			ExpectedResult: true,
			IsError:        false,
			Worker:         DefaultWorker,
			Target:         DefaultTarget,
		},
	}

	for _, test := range tc {
		t.Run(test.Name, func(t *testing.T) {
			// Open our jsonFile
			jsonFile, err := os.Open(test.TCFile)
			// if we os.Open returns an error then handle it
			if !assert.Nil(t, err) {
				return
			}

			// defer the closing of our jsonFile so that we can parse it later on
			defer jsonFile.Close()

			// read our opened xmlFile as a byte array.
			byteValue, _ := ioutil.ReadAll(jsonFile)

			var ruleMap struct {
				Data []Rule `json:"data"`
			}

			err = json.Unmarshal(byteValue, &ruleMap)
			if !assert.Nil(t, err) {
				return
			}

			e := New()
			e.Set(test.Facts, ruleMap.Data, test.RuleFunction)
			res, errs := e.Run(test.Target, test.Worker)
			if test.IsError {
				assert.NotNil(t, errs)
			} else {
				if assert.NotNil(t, res) {
					assert.Equal(t, test.ExpectedResult, res)
				}
			}
		})
	}
}

func BenchmarkRun(b *testing.B) {
	tc := []struct {
		Name           string
		TCFile         string
		Facts          map[string]interface{}
		RuleFunction   map[string]RuleFunction
		Target         string
		Worker         int
		ExpectedResult interface{}
		IsError        bool
	}{
		{
			Name:   "DefaultWorker",
			TCFile: "./test/tc1.json",
			Facts: map[string]interface{}{
				"account_partner": "hello",
				"account_region":  "ID",
				"flight_type":     "free",
			},
			ExpectedResult: true,
			IsError:        false,
			Worker:         DefaultWorker,
			Target:         DefaultTarget,
		},
		{
			Name:   "1 Worker",
			TCFile: "./test/tc1.json",
			Facts: map[string]interface{}{
				"account_partner": "hello",
				"account_region":  "ID",
				"flight_type":     "free",
			},
			ExpectedResult: true,
			IsError:        false,
			Worker:         1,
			Target:         DefaultTarget,
		},
		{
			Name:   "Equal CPU Count",
			TCFile: "./test/tc1.json",
			Facts: map[string]interface{}{
				"account_partner": "hello",
				"account_region":  "ID",
				"flight_type":     "free",
			},
			ExpectedResult: true,
			IsError:        false,
			Worker:         runtime.NumCPU(),
			Target:         DefaultTarget,
		},
		{
			Name:   "Double CPU Count",
			TCFile: "./test/tc1.json",
			Facts: map[string]interface{}{
				"account_partner": "hello",
				"account_region":  "ID",
				"flight_type":     "free",
			},
			ExpectedResult: true,
			IsError:        false,
			Worker:         runtime.NumCPU() * 2,
			Target:         DefaultTarget,
		},
	}

	for _, test := range tc {
		b.Run(test.Name, func(b *testing.B) {
			// Open our jsonFile
			jsonFile, _ := os.Open(test.TCFile)
			// if we os.Open returns an error then handle it

			// defer the closing of our jsonFile so that we can parse it later on
			defer jsonFile.Close()

			// read our opened xmlFile as a byte array.
			byteValue, _ := ioutil.ReadAll(jsonFile)

			var ruleMap struct {
				Data []Rule `json:"data"`
			}

			json.Unmarshal(byteValue, &ruleMap)

			e := New()
			e.Set(test.Facts, ruleMap.Data, test.RuleFunction)
			for i := 0; i < b.N; i++ {
				e.Run(test.Target, test.Worker)
			}
		})
	}
}
