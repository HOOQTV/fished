package fished

import (
	"sync"

	"github.com/knetic/govaluate"
)

// Config ...
type Config struct {
	Worker         int
	WorkerPoolSize int
	Cache          bool
}

// Engine ...
type Engine struct {
	Rules        []Rule
	rf           map[string]govaluate.ExpressionFunction
	Jobs         chan int
	Config       *Config
	work         sync.WaitGroup
	wmMutex      sync.RWMutex
	runMutex     sync.Mutex
	factsMutex   sync.RWMutex
	err          []error
	wm           map[string]interface{}
	workingRules map[string][]int
}

// Rule ...
type Rule struct {
	Output      string   `json:"output"`
	Input       []string `json:"input"`
	Expression  string   `json:"expression"`
	ee          *govaluate.EvaluableExpression
	facts       map[string]interface{}
	result      interface{}
	hasExecuted bool
}

// RuleRaw ...
type RuleRaw struct {
	Data []Rule `json:"data"`
}

// RuleFunction ...
type RuleFunction func(arguments ...interface{}) (interface{}, error)

var workerPoolSize = 10

// New ...
func New(c *Config) *Engine {
	cfg := &Config{
		Worker:         1,
		WorkerPoolSize: 20,
		Cache:          false,
	}
	if c != nil {
		if c.Worker > 0 {
			cfg.Worker = c.Worker
		}
		if c.WorkerPoolSize > (0 + cfg.Worker) {
			cfg.WorkerPoolSize = c.WorkerPoolSize
		}
		cfg.Cache = c.Cache
	}
	e := &Engine{
		Config: cfg,
		err:    []error{},
	}
	return e
}

// SetFacts ...
func (e *Engine) SetFacts(f map[string]interface{}) {
	e.wm = make(map[string]interface{})
	for i, v := range f {
		e.wm[i] = v
	}
}

// SetRules ...
func (e *Engine) SetRules(r []Rule) {
	e.Rules = make([]Rule, len(r))
	copy(e.Rules, r)

	e.workingRules = make(map[string][]int)
	for i, rule := range r {
		for _, input := range rule.Input {
			if e.workingRules[input] == nil {
				e.workingRules[input] = []int{i}
			} else {
				e.workingRules[input] = append(e.workingRules[input], i)
			}
		}
	}
}

// SetRuleFunction ...
func (e *Engine) SetRuleFunction(rf map[string]RuleFunction) {
	e.rf = make(map[string]govaluate.ExpressionFunction)
	for i, f := range rf {
		e.rf[i] = govaluate.ExpressionFunction(f)
	}
}

// Run ...
func (e *Engine) Run(target ...string) (interface{}, []error) {
	e.runMutex.Lock()
	defer e.runMutex.Unlock()

	e.Jobs = make(chan int, e.Config.WorkerPoolSize)

	var wg sync.WaitGroup

	e.createAgenda()
	for i := 0; i < e.Config.Worker; i++ {
		wg.Add(1)
		go e.worker(&wg)
	}

	e.watcher()
	wg.Wait()
	res := "result_end"
	if len(target) == 1 {
		res = target[0]
	}

	return e.wm[res], e.err
}

func (e *Engine) watcher() {
	e.work.Wait()
	close(e.Jobs)

	for _, rule := range e.Rules {
		rule.hasExecuted = false
	}
}

func (e *Engine) worker(wg *sync.WaitGroup) {
	for job := range e.Jobs {
		e.eval(job)
		e.work.Done()
	}
	wg.Done()
}

// eval will evaluate current rule.
func (e *Engine) eval(index int) {
	if e.Rules[index].Output != "" && !e.Rules[index].hasExecuted {
		e.Rules[index].hasExecuted = true
		if e.Rules[index].ee == nil {
			re, err := govaluate.NewEvaluableExpressionWithFunctions(e.Rules[index].Expression, e.rf)
			if err != nil {
				e.err = append(e.err, err)
				return
			}
			e.Rules[index].ee = re
		}

		if e.Rules[index].result == nil || !e.Config.Cache {
			e.factsMutex.RLock()
			result, err := e.Rules[index].ee.Evaluate(e.Rules[index].facts)
			e.factsMutex.RUnlock()
			if err != nil {
				e.err = append(e.err, err)
				return
			}

			resBool, ok := result.(bool)
			if (result != nil && !ok) || resBool {
				e.Rules[index].result = result
			}

		}

		if e.Rules[index].result != nil {
			e.wmMutex.Lock()
			e.wm[e.Rules[index].Output] = e.Rules[index].result
			e.wmMutex.Unlock()
			e.updateAgenda(e.Rules[index].Output)
		}
	}
}

// Add jobs base on current working memory attribute
func (e *Engine) updateAgenda(input string) {
	rules := e.workingRules[input]
	for _, i := range rules {
		rule := e.Rules[i]
		validInput := 0
		e.wmMutex.RLock()
		for attribute, value := range e.wm {
			for _, input := range rule.Input {
				if input == attribute {
					validInput++
					e.factsMutex.Lock()
					if rule.facts == nil {
						rule.facts = make(map[string]interface{})
					}
					if rule.facts[attribute] == nil || !e.Config.Cache {
						rule.facts[attribute] = value
					}
					e.factsMutex.Unlock()
				}
			}
		}
		e.wmMutex.RUnlock()
		if validInput == len(rule.Input) && validInput != 0 {
			e.work.Add(1)
			e.Jobs <- i
		}
		e.Rules[i] = rule
	}
}

// initialize jobs
func (e *Engine) createAgenda() {
	for attribute := range e.wm {
		e.updateAgenda(attribute)
	}
}
