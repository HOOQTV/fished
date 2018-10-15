package fished

import (
	"runtime"
	"sync"
	"time"

	"github.com/knetic/govaluate"
	"github.com/patrickmn/go-cache"
)

var (
	//DefaultTarget is the default target facts
	DefaultTarget = "result_end"

	// DefaultWorker is the default worker for Engine
	DefaultWorker = 0
)

type (
	// Engine core of the machine
	Engine struct {
		InitialFacts  map[string]interface{}
		Rules         []Rule
		RuleFunctions map[string]govaluate.ExpressionFunction
		RuleCache     *cache.Cache
		RunLock       sync.RWMutex
	}

	// Rule is struct for rule in fished
	Rule struct {
		Input      []string `json:"input"`
		Output     string   `json:"output"`
		Expression string   `json:"expression"`
	}

	// RuleFunction if type defined for rule function
	RuleFunction func(...interface{}) (interface{}, error)

	// Runtime is an struct for each time Engine.Run() is called
	Runtime struct {
		Facts      map[string]interface{}
		JobCh      chan *Job
		ResultCh   chan *EvalResult
		UsedRule   map[int]struct{}
		FactsMutex sync.RWMutex
		WorkerWg   sync.WaitGroup
	}

	// Job struct
	Job struct {
		Output           string
		ParsedExpression *govaluate.EvaluableExpression
	}

	// EvalResult is evaluation Result
	EvalResult struct {
		Key   string
		Value interface{}
		Error error
	}
)

// New will create new engine
func New() *Engine {
	c := cache.New(24*time.Hour, 1*time.Hour)

	return &Engine{
		RuleCache: c,
	}
}

// Set all of engine attibutes in one single function
func (e *Engine) Set(facts map[string]interface{}, rules []Rule, ruleFunction map[string]RuleFunction) error {
	var err error

	err = e.SetFacts(facts)
	if err != nil {
		return err
	}

	err = e.SetRules(rules)
	if err != nil {
		return err
	}

	err = e.SetRuleFunctions(ruleFunction)
	if err != nil {
		return err
	}

	return nil
}

// SetFacts will set current engine with initial facts (replace the old one)
func (e *Engine) SetFacts(facts map[string]interface{}) error {
	e.RunLock.Lock()
	defer e.RunLock.Unlock()
	e.InitialFacts = make(map[string]interface{})

	for key, value := range facts {
		e.InitialFacts[key] = value
	}
	return nil
}

// SetRules will set current engine with rules
func (e *Engine) SetRules(rules []Rule) error {
	e.RunLock.Lock()
	defer e.RunLock.Unlock()
	e.Rules = make([]Rule, len(rules))
	copy(e.Rules, rules)
	e.RuleCache.Flush()
	return nil
}

// SetRuleFunctions will set current engine with Expression Functions
func (e *Engine) SetRuleFunctions(ruleFunctions map[string]RuleFunction) error {
	e.RunLock.Lock()
	defer e.RunLock.Unlock()
	e.RuleFunctions = make(map[string]govaluate.ExpressionFunction)

	for key, value := range ruleFunctions {
		e.RuleFunctions[key] = govaluate.ExpressionFunction(value)
	}
	return nil
}

// RunDefault will execute run with default parameneter
func (e *Engine) RunDefault() (interface{}, []error) {
	return e.Run(DefaultTarget, DefaultWorker)
}

// Run will execute rule and facts to get the result
func (e *Engine) Run(target string, worker int) (interface{}, []error) {
	var workerSize int
	var endTarget string
	var errs []error

	e.RunLock.RLock()
	defer e.RunLock.RUnlock()

	if target == DefaultTarget {
		endTarget = DefaultTarget
	} else {
		endTarget = target
	}

	if worker == DefaultWorker {
		numCPU := runtime.NumCPU()
		if numCPU <= 2 {
			workerSize = 1
		} else {
			workerSize = runtime.NumCPU() - 1
		}
	} else {
		workerSize = worker
	}

	if workerSize <= 0 {
		workerSize = 1
	}

	facts := make(map[string]interface{})
	for key, value := range e.InitialFacts {
		facts[key] = value
	}

	r := &Runtime{
		Facts:    facts,
		JobCh:    make(chan *Job, len(e.Rules)),
		ResultCh: make(chan *EvalResult, len(e.Rules)),
		UsedRule: make(map[int]struct{}),
	}

	r.WorkerWg.Add(workerSize)
	for i := 0; i < workerSize; i++ {
		go func() {
			defer r.WorkerWg.Done()
			for job := range r.JobCh {
				r.Evaluate(job, r.ResultCh)
			}
		}()
	}

	for {
		var jobLength int
		var parseRuleError bool
		for i := range e.Rules {
			// Check if the rule already been executed
			if _, ok := r.UsedRule[i]; ok {
				continue
			}

			// copy rule into context
			rule := e.Rules[i]

			// Verify if rule has met input requirement
			inputLen := len(rule.Input)
			if inputLen > 0 {
				var ValidInput int
				for _, input := range rule.Input {
					if _, ok := r.Facts[input]; ok {
						ValidInput++
					}
				}
				if inputLen != ValidInput {
					continue
				}
			}

			// Check cache for parsed rule
			parsedExpression, ok := e.RuleCache.Get(rule.Expression)

			// if not exist in cache then parse rule
			if !ok {
				var err error
				parsedExpression, err = govaluate.NewEvaluableExpressionWithFunctions(rule.Expression, e.RuleFunctions)
				if err != nil {
					if errs == nil {
						errs = make([]error, 0)
					}
					errs = append(errs, err)
					parseRuleError = true
					break
				}

				err = e.RuleCache.Add(rule.Expression, parsedExpression, cache.DefaultExpiration)
				if err != nil {
					if errs == nil {
						errs = make([]error, 0)
					}
					errs = append(errs, err)
					parseRuleError = true
					break
				}
			}

			j := &Job{
				ParsedExpression: parsedExpression.(*govaluate.EvaluableExpression),
				Output:           rule.Output,
			}
			r.UsedRule[i] = struct{}{}
			r.JobCh <- j
			jobLength++
		}

		if jobLength == 0 || parseRuleError {
			r.PrepareExit()
			break
		}

		for jobs := 0; jobs < jobLength; jobs++ {
			evalResult := <-r.ResultCh
			if evalResult.Error != nil {
				if errs == nil {
					errs = make([]error, 0)
				}
				errs = append(errs, evalResult.Error)
				continue
			}
			if evalResult.Value != nil {
				r.FactsMutex.Lock()
				r.Facts[evalResult.Key] = evalResult.Value
				r.FactsMutex.Unlock()
			}
		}
	}

	r.WorkerWg.Wait()
	return r.Facts[endTarget], errs
}

// Evaluate will evaluate each job in runtime
func (r *Runtime) Evaluate(job *Job, result chan<- *EvalResult) {
	evalResult := &EvalResult{
		Key: job.Output,
	}

	r.FactsMutex.RLock()
	res, err := job.ParsedExpression.Evaluate(r.Facts)
	r.FactsMutex.RUnlock()
	if err != nil {
		evalResult.Error = err
	}
	evalResult.Value = res
	result <- evalResult
}

// PrepareExit the runtime and close
func (r *Runtime) PrepareExit() {
	close(r.JobCh)
	close(r.ResultCh)
}
