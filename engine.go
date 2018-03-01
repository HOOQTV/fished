package fished

import (
	"fmt"
	"sync"

	"github.com/knetic/govaluate"
)

// Engine ...
type Engine struct {
	Facts         map[string]interface{}
	Rules         []Rule
	RuleFunctions map[string]RuleFunction
	rf            map[string]govaluate.ExpressionFunction
	Jobs          chan int
	Worker        int
	work          sync.WaitGroup
	wmLock        sync.Mutex
	planLock      sync.Mutex
	err           []error
	wm            map[string]interface{}
	workingRules  map[string][]int
}

// Rule ...
type Rule struct {
	Output     string   `json:"output"`
	Input      []string `json:"input"`
	Expression string   `json:"expression"`
}

// RuleRaw ...
type RuleRaw struct {
	Data []Rule `json:"data"`
}

// RuleFunction ...
type RuleFunction func(arguments ...interface{}) (interface{}, error)

var workerPoolSize = 10

// New ...
func New(worker int) *Engine {
	e := &Engine{
		Jobs:          make(chan int, worker*workerPoolSize),
		Worker:        worker,
		Facts:         make(map[string]interface{}),
		RuleFunctions: make(map[string]RuleFunction),
		err:           []error{},
	}
	return e
}

// Run ...
func (e *Engine) Run(target ...string) (interface{}, []error) {
	var wg sync.WaitGroup
	e.wm = make(map[string]interface{})
	for i, v := range e.Facts {
		e.wm[i] = v
	}

	e.workingRules = make(map[string][]int)
	for i, rule := range e.Rules {
		for _, input := range rule.Input {
			if e.workingRules[input] == nil {
				e.workingRules[input] = []int{i}
			} else {
				e.workingRules[input] = append(e.workingRules[input], i)
			}
		}
	}

	e.rf = make(map[string]govaluate.ExpressionFunction)
	for i, f := range e.RuleFunctions {
		e.rf[i] = govaluate.ExpressionFunction(f)
	}

	e.createAgenda()
	for i := 0; i < e.Worker; i++ {
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
	if e.Rules[index].Output != "" {
		re, err := govaluate.NewEvaluableExpressionWithFunctions(e.Rules[index].Expression, e.rf)
		if err != nil {
			e.err = append(e.err, err)
			return
		}
		e.wmLock.Lock()
		res, err := re.Evaluate(e.wm)
		e.wmLock.Unlock()
		if err != nil {
			e.err = append(e.err, err)
			return
		}

		e.wmLock.Lock()
		e.wm[e.Rules[index].Output] = res
		e.wmLock.Unlock()
		e.updateAgenda(e.Rules[index].Output)
	}
}

// Add jobs base on current working memory attribute
func (e *Engine) updateAgenda(input string) {
	e.planLock.Lock()
	defer e.planLock.Unlock()
	e.wmLock.Lock()
	defer e.wmLock.Unlock()

	rules := e.workingRules[input]
	for _, i := range rules {
		rule := e.Rules[i]
		validInput := 0
		for attribute := range e.wm {
			for _, input := range rule.Input {
				if input == attribute {
					validInput++
				}
			}
		}
		if validInput == len(rule.Input) && validInput != 0 {
			fmt.Println("Output target:", rule.Output, "Added")
			e.work.Add(1)
			e.Jobs <- i
		}
	}
	delete(e.workingRules, input)
}

// initialize jobs
func (e *Engine) createAgenda() {
	for attribute := range e.wm {
		e.updateAgenda(attribute)
	}
}
