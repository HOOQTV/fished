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

	e.workingRules = make([]Rule, len(e.Rules))
	copy(e.workingRules, e.Rules)

	e.createAgenda()
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
		if e.eval(job.CurRule) {
			e.createAgenda()
		}
		e.work.Done()
	}
	wg.Done()
}

// eval return true or false that will invoke need to update agenda or not.
func (e *Engine) eval(r Rule) bool {
	fmt.Println(r.Output, "called")
	re, _ := govaluate.NewEvaluableExpressionWithFunctions(r.Rule, e.RuleFunctions)
	// fmt.Println("Rule Memory:", r)
	// fmt.Println("Working Memory:", workingMemory)
	valid, _ := re.Evaluate(e.wm)
	fmt.Println(r.Output, "result: ", valid)

	if valid != nil && valid.(bool) {
		ve, err := govaluate.NewEvaluableExpressionWithFunctions(r.Value, e.RuleFunctions)
		if err == nil {
			res, _ := ve.Evaluate(nil)

			if r.Output != "" {
				e.wmLock.Lock()
				e.wm[r.Output] = res
				e.wmLock.Unlock()
				return true
			}
		}
	}
	return false
}

func (e *Engine) createAgenda() {
	e.planLock.Lock()
	defer e.planLock.Unlock()
	e.wmLock.Lock()
	defer e.wmLock.Unlock()

	i := 0
	for i < len(e.workingRules) {
		rule := e.workingRules[i]
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
			j := &Job{
				CurRule: rule,
			}
			e.work.Add(1)
			e.Jobs <- *j

			e.workingRules[i] = e.workingRules[len(e.workingRules)-1]
			e.workingRules[len(e.workingRules)-1] = Rule{}
			e.workingRules = e.workingRules[:len(e.workingRules)-1]
		} else {
			i++
		}
	}
}
