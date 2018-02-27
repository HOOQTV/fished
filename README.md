# fished
Fish answers from rules and facts

# Install
```
go get -u github.com/jonathansudibya/fished
```

or using dep
```
dep ensure -add github.com/jonathansudibya/fished
```

# Structs
Rule :
```
type Rule struct {
	Output string   `json:"output"`
	Input  []string `json:"input"`
	Rule   string   `json:"rule"`
	Value  string   `json:"value"`
}
```
Engine:
```
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

# Credits
This project is powered by 
- https://github.com/Knetic/govaluate. Please check if you need an abritrary expression checker!
- https://github.com/json-iterator/go. One of (or the fastest) drop-in replacement for standart json library.
- https://github.com/stretchr/testify. A nice and clean testing library. Helping me a lot for better test results.

