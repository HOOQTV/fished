package fished

import (
	"encoding/hex"
	"runtime"
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
		rules         sync.Map
		usedRule      map[string]interface{}
		WorkerSize    int
		workerPool    chan chan job
		jobQueue      chan job
		workers       []worker
		exit          chan bool
	}

	// RuleFunction ..
	RuleFunction func(map[string]interface{}) (interface{}, error)

	// job ...
	job struct {
		State      map[string]interface{}
		Expression *conditions.Expr
	}

	// worker ...
	worker struct {
		WorkerPool chan chan job
		JobChannel chan job
		quit       chan bool
	}
)

var (
	// RuleBook ...
	RuleBook sync.Map
)

func newWorker(workerPool chan chan job) worker {
	return worker{
		WorkerPool: workerPool,
		JobChannel: make(chan job),
		quit:       make(chan bool),
	}
}

func (w *worker) Start() {
	go func() {
		for {
			w.WorkerPool <- w.JobChannel

			select {
			case job := <-w.JobChannel:
				evaluate(job)
				updateAgenda()
			case <-w.quit:
				return
			}
		}
	}()
}

func (w *worker) Stop() {
	go func() {
		w.quit <- true
	}()
}

// New ...
func New(worker int) (*Engine, error) {
	ws := worker
	if ws <= 0 {
		ws = runtime.NumCPU()
	}

	pool := make(chan chan job, ws)

	return &Engine{
		initialState: make(map[string]interface{}),
		WorkerSize:   ws,
		workerPool:   pool,
	}, nil
}

// Run ...
func (e *Engine) Run() {
	e.workers = make([]worker, e.WorkerSize)

	for i := 0; i < e.WorkerSize; i++ {
		e.workers[i] = newWorker(e.workerPool)
		e.workers[i].Start()
	}

	go e.dispatch()
}

func (e *Engine) dispatch() {
	for {
		select {
		case j := <-e.jobQueue:
			go func(j job) {
				jobChannel := <-e.workerPool
				jobChannel <- j
			}(j)
		case <-e.exit:
			for i := 0; i < e.WorkerSize; i++ {
				e.workers[i].Stop()
			}
			return
		}
	}
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
	e.usedRule = make(map[string]interface{})
	for _, rule := range rules {
		wg.Add(1)
		go func(rule Rule) {
			ruleHash := hex.EncodeToString(getMD5Hash([]byte(rule.Expression)))
			rule.ID = ruleHash

			parser := conditions.NewParser(strings.NewReader(rule.Expression))
			parsedExpression, _ := parser.Parse()

			RuleBook.Store(rule.ID, &parsedExpression)

			e.rules.Store(rule.ID, rule)
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

func evaluate(j job) error {
	return nil
}

func updateAgenda() error {
	return nil
}
