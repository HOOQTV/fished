package main

import (
	"fmt"
	"sync"

	"github.com/knetic/govaluate"
)

// Engine ...
type Engine struct {
	Facts         map[string]interface{}
	wm            map[string]interface{}
	Rules         []Rule
	workingRules  []Rule
	RuleFunctions map[string]govaluate.ExpressionFunction
	Jobs          chan Job
	Worker        int
	work          sync.WaitGroup
	wmLock        sync.Mutex
	planLock      sync.Mutex
}

// Rule ...
type Rule struct {
	Output string   `json:"output"`
	Input  []string `json:"input"`
	Rule   string   `json:"rule"`
	Value  string   `json:"value"`
}

// Job ...
type Job struct {
	CurRule Rule
}

var workerPoolSize = 10

// New ...
func New(worker int) *Engine {
	e := &Engine{
		Jobs:   make(chan Job, worker*workerPoolSize),
		Worker: worker,
	}
	return e
}

// Run ...
func (e *Engine) Run() interface{} {
	var wg sync.WaitGroup
	e.wm = make(map[string]interface{})
	for i, v := range e.Facts {
		e.wm[i] = v
	}
	e.createAgenda(e.Rules, e.wm)
	for i := 0; i < e.Worker; i++ {
		wg.Add(1)
		go e.worker(&wg)
	}
	e.watcher()
	wg.Wait()

	return e.wm["result_end"]
}

func (e *Engine) watcher() {
	e.work.Wait()
	close(e.Jobs)
}

func (e *Engine) worker(wg *sync.WaitGroup) {
	for job := range e.Jobs {
		if e.eval(job.CurRule, e.wm, e.RuleFunctions) {
			e.planLock.Lock()
			e.createAgenda(e.workingRules, e.wm)
			e.planLock.Unlock()
		}
		e.work.Done()
	}
	wg.Done()
}

// eval return true or false that will invoke need to update agenda or not.
func (e *Engine) eval(r Rule, workingMemory map[string]interface{}, f map[string]govaluate.ExpressionFunction) bool {
	fmt.Println(r.Output, "called")
	re, _ := govaluate.NewEvaluableExpressionWithFunctions(r.Rule, f)
	e.wmLock.Lock()
	// fmt.Println("Rule Memory:", r)
	// fmt.Println("Working Memory:", workingMemory)
	valid, _ := re.Evaluate(workingMemory)
	e.wmLock.Unlock()
	fmt.Println(r.Output, "result: ", valid)

	if valid != nil && valid.(bool) {
		ve, err := govaluate.NewEvaluableExpressionWithFunctions(r.Value, f)
		if err == nil {
			res, _ := ve.Evaluate(nil)

			if r.Output != "" {
				e.wmLock.Lock()
				workingMemory[r.Output] = res
				e.wmLock.Unlock()
				return true
			}
		}
	}
	return false
}

func (e *Engine) createAgenda(rules []Rule, workingMemory map[string]interface{}) {
	fmt.Println("Rule length", len(rules))

	e.wmLock.Lock()
	i := 0
	for i < len(rules) {
		rule := rules[i]
		validInput := 0
		for attribute := range workingMemory {
			for _, input := range rule.Input {
				if input == attribute {
					validInput++
				}
			}
		}
		if validInput == len(rule.Input) && validInput != 0 {
			fmt.Println("Output target:", rule.Output, "Added")
			j := &Job{
				CurRule: rule,
			}
			e.work.Add(1)
			e.Jobs <- *j

			rules[i] = rules[len(rules)-1]
			rules[len(rules)-1] = Rule{}
			rules = rules[:len(rules)-1]
		} else {
			i++
		}
	}
	e.wmLock.Unlock()
	e.workingRules = rules
}
