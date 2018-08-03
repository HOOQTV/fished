package fished

import (
	"encoding/hex"
	"strings"
	"sync"

	"github.com/oleksandr/conditions"
)

type (
	// Rule ...
	Rule struct {
		ID         string
		Input      []string
		Expression string
		Output     map[string]RuleOutput
	}

	// RuleOutput ...
	RuleOutput struct {
		Type              string
		Value             interface{}
		Function          *string
		Parameter         *[]string
		ConstantParameter *[]string
	}

	// ruleObject ...
	ruleObject struct {
		Input    map[string]interface{}
		RuleHash []byte
	}

	// Engine ...
	Engine struct {
		ruleFunctions map[string]RuleFunction
		initialState  map[string]interface{}
		workingMemory sync.Map
		rules         map[string]Rule
		usedRule      map[string]interface{}
		Exit          bool
		working       sync.WaitGroup
	}

	// RuleFunction ..
	RuleFunction func(map[string]interface{}) (interface{}, error)
)

var (
	// RuleBook ...
	RuleBook sync.Map
)

// New ...
func New() (*Engine, error) {
	return &Engine{
		initialState: make(map[string]interface{}),
		rules:        make(map[string]Rule),
	}, nil
}

// SetFacts ...
func (e *Engine) SetFacts(facts map[string]interface{}) error {
	for key, value := range facts {
		e.initialState[key] = value
	}
	return nil
}

// SetRules ...
func (e *Engine) SetRules(rules []Rule) error {
	var wg sync.WaitGroup
	e.rules = make(map[string]Rule)
	e.usedRule = make(map[string]interface{})
	for _, rule := range rules {
		wg.Add(1)
		go func(rule Rule) {
			ruleHash := hex.EncodeToString(getMD5Hash([]byte(rule.Expression)))
			rule.ID = ruleHash

			parser := conditions.NewParser(strings.NewReader(rule.Expression))
			parsedExpression, _ := parser.Parse()

			RuleBook.Store(ruleHash, parsedExpression)

			e.rules[ruleHash] = rule
			wg.Done()
		}(rule)
	}
	wg.Wait()
	return nil
}

// SetRuleFunctions ...
func (e *Engine) SetRuleFunctions(rf map[string]RuleFunction) error {
	for key, value := range rf {
		e.ruleFunctions[key] = value
	}

	return nil
}

// Run ...
func (e *Engine) Run() (interface{}, error) {
	task := make(chan string)
	queue := make(chan string)
	done := make(chan bool)

	go e.taskMaster(task, queue, done)
	go e.dispatcher(task, queue, done)

	// Copy initial state to working memory
	var wg sync.WaitGroup
	for key, value := range e.initialState {
		wg.Add(1)
		go func(key string, value interface{}) {
			e.workingMemory.Store(key, value)
			task <- key
			wg.Done()
		}(key, value)
	}
	wg.Wait()

	result, _ := e.workingMemory.Load("result_end")

	// Clean up
	e.workingMemory.Range(func(key, value interface{}) bool {
		e.workingMemory.Delete(key)
		return true
	})
	return result, nil
}

func (e *Engine) taskMaster(task, queue chan string, done chan bool) {
	for {
		select {
		case <-task:
			e.addAgenda(queue, done)
		case <-done:
			close(task)
			close(queue)
			close(done)
			return
		}
	}
}

func (e *Engine) addAgenda(queue chan string, done chan bool) {
	addingAgenda := false
	for _, v := range e.rules {
		if _, ok := e.usedRule[v.ID]; !ok {
			validInput := 0
			for i := 0; i < len(v.Input); i++ {
				if _, ok := e.workingMemory.Load(v.Input[i]); ok {
					validInput++
				}
			}
			if validInput == len(v.Input) {
				queue <- v.ID
				e.usedRule[v.ID] = nil
				addingAgenda = true
			}
		}
	}
	if !addingAgenda {
		e.working.Wait()
		if e.Exit {
			done <- true
			return
		}
		e.Exit = true
	} else if e.Exit {
		e.Exit = false
	}
}

func (e *Engine) dispatcher(task, queue chan string, done chan bool) {
	for job := range queue {
		go e.eval(job, task, done)
	}
}

func (e *Engine) eval(job string, task chan string, done chan bool) {
	e.working.Add(1)
	defer e.working.Done()

	rule := e.rules[job]
	facts := make(map[string]interface{})

	for _, v := range rule.Input {
		facts[v], _ = e.workingMemory.Load(v)
	}

	parsedExpression, _ := RuleBook.Load(rule.ID)
	correct, err := conditions.Evaluate(parsedExpression.(conditions.Expr), facts)
	if err != nil {
		done <- true
	}

	if correct {
		for k, v := range rule.Output {
			if v.Value != nil {
				e.workingMemory.Store(k, v)
			} else if v.Function != nil {
				f := e.ruleFunctions[*v.Function]
				inputs := make(map[string]interface{})
				for _, key := range *v.Parameter {
					inputs[key], _ = e.workingMemory.Load(key)
				}

				result, _ := f(inputs)

				if v.Type == "single" {
					e.workingMemory.Store(k, result)
					task <- k
				} else if v.Type == "map" {
					if mapResult, ok := result.(map[string]interface{}); ok {
						for resKey, resVal := range mapResult {
							newKey := k + "_" + resKey
							e.workingMemory.Store(newKey, resVal)
							task <- newKey
						}
					}
					// TODO : ADD ERROR HANDLING
				}
			}
			if k == "result_end" {
				done <- true
				break
			}
		}
	}
}
