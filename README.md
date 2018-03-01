

# fished
Fish answers from rules and facts

# Install
```
$ go get -u github.com/jonathansudibya/fished
```

or using dep
```
$ dep ensure -add github.com/jonathansudibya/fished
```

# Structs
Rule :
```go
type Rule struct {
	Output 			string   `json:"output"`
	Input  			[]string `json:"input"`
	Expression   	string   `json:"expression"`
}
```
Engine:
```go
type Engine struct {
    Facts         map[string]interface{}
	Rules         []Rule
	RuleFunctions map[string]govaluate.ExpressionFunction
	Jobs          chan int
    ...
}
```

# Example
Please see in test file especially `TestRun()` function.
```go
package main

import "github.com/jonathansudibya/fished"

func main() {
	e := fished.New(10)
	e.Rules = []Rule{
		Rule{
			Input: []string{"hello"},
			Output: "result_end",
			Expression: "hello == 'world'",
		}
	}
	e.Facts["hello"] = "world"
	res, errs := e.Run()

	fmt.Println(res) // will result true
}
```

# Credits
This project is powered by 
- https://github.com/Knetic/govaluate. Please check if you need an abritrary expression checker!
- https://github.com/json-iterator/go. One of (or the fastest) drop-in replacement for standart json library.
- https://github.com/stretchr/testify. A nice and clean testing library. Helping me a lot for better test results.

# LICENSE
See LICENSE.md
