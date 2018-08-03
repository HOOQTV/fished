package fished

import (
	"encoding/hex"
	"strings"
	"sync"
	"time"

	"github.com/allegro/bigcache"
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
		workingMemory *bigcache.BigCache
		rules         map[string]Rule
		usedRule      map[string]interface{}
	}

	// RuleFunction ..
	RuleFunction func(map[string]interface{}) (interface{}, error)
)

var (
	// RuleBook ...
	RuleBook map[string]conditions.Expr
)

func init() {
	RuleBook = make(map[string]conditions.Expr)
}

// New ...
func New() (*Engine, error) {
	// TODO: Make eviction time configurable
	wm, err := bigcache.NewBigCache(bigcache.DefaultConfig(10 * time.Minute))
	if err != nil {
		return nil, err
	}

	return &Engine{
		initialState:  make(map[string]interface{}),
		workingMemory: wm,
		rules:         make(map[string]Rule),
	}, nil
}

// SetFacts ...
func (e *Engine) SetFacts(facts map[string]interface{}) error {
	for k, v := range facts {
		e.initialState[k] = v
	}
	return nil
}

// SetRules ...
func (e *Engine) SetRules(rules []Rule) error {
	var wg sync.WaitGroup
	e.rules = make(map[string]Rule)
	e.usedRule = make(map[string]interface{})
	for _, v := range rules {
		wg.Add(1)
		go func() {
			ruleHash := hex.EncodeToString(getMD5Hash([]byte(v.Expression)))
			v.ID = ruleHash

			parser := conditions.NewParser(strings.NewReader(v.Expression))
			parsedExpression, _ := parser.Parse()

			if _, ok := RuleBook[ruleHash]; !ok {
				RuleBook[ruleHash] = parsedExpression
			}
			e.rules[ruleHash] = v
			wg.Done()
		}()
	}
	wg.Wait()
	return nil
}

// SetRuleFunctions ...
func (e *Engine) SetRuleFunctions(rf map[string]RuleFunction) error {
	for k, v := range rf {
		e.ruleFunctions[k] = v
	}

	return nil
}

// Run ...
func (e *Engine) Run() error {
	task := make(chan string)
	queue := make(chan string)
	done := make(chan bool)

	go e.taskMaster(task, queue, done)
	go e.dispatcher(task, queue, done)

	// Copy initial state to working memory
	var wg sync.WaitGroup
	for k, v := range e.initialState {
		wg.Add(1)
		go func() {
			b, _ := getBytes(v)
			e.workingMemory.Set(k, b)
			task <- k
			wg.Done()
		}()
	}
	wg.Wait()

	return nil
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
				if _, err := e.workingMemory.Get(v.Input[i]); err != nil {
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
		done <- true
	}
}

func (e *Engine) dispatcher(task, queue chan string, done chan bool) {
	for job := range queue {
		go e.eval(job, task, done)
	}
}

func (e *Engine) eval(job string, task chan string, done chan bool) {
	rule := e.rules[job]
	facts := make(map[string]interface{})

	for _, v := range rule.Input {
		var fact interface{}
		factBytes, _ := e.workingMemory.Get(v)
		getInterface(factBytes, fact)
		facts[v] = fact
	}

	parsedExpression := RuleBook[rule.ID]
	correct, err := conditions.Evaluate(parsedExpression, facts)
	if err != nil {
		done <- true
	}

	if correct {
		for k, v := range rule.Output {
			if v.Value != nil {
				outputBytes, _ := getBytes(v.Value)
				e.workingMemory.Set(k, outputBytes)
			} else if v.Function != nil {
				f := e.ruleFunctions[*v.Function]
				inputs := make(map[string]interface{})
				for _, key := range *v.Parameter {
					paramBytes, _ := e.workingMemory.Get(key)
					var param interface{}
					getInterface(paramBytes, param)
					inputs[key] = param
				}

				result, _ := f(inputs)

				if v.Type == "single" {
					resultBytes, _ := getBytes(result)
					e.workingMemory.Set(k, resultBytes)
					task <- k
				} else if v.Type == "map" {
					if mapResult, ok := result.(map[string]interface{}); ok {
						for resKey, resVal := range mapResult {
							newKey := k + "_" + resKey
							resultBytes, _ := getBytes(resVal)
							e.workingMemory.Set(newKey, resultBytes)
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
