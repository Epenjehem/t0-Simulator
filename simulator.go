// Package t0simulator provides an instrument to simulating time out budgeting
package t0simulator

import (
	"context"
	"fmt"
	"io"
	"os"
	"text/tabwriter"
	"time"
)

const rowFormat = "%s\t%v\t%v\t\n"

// Proccess denotes an interface of simulated process
type Proccess interface {
	Run(ctx context.Context, w io.Writer)
	IsExecuted() bool
	String() string
}

// Function denotes a function that will be run in simulator
type Function struct {
	name       string
	isExecuted bool
}

// NewFunction return a new Function
func NewFunction(name string) Function {
	return Function{
		name: name,
	}
}

// WithTimeout returns a simulated function that will be run with context timeout
func (f Function) WithTimeout(timeout int) *FunctionWithTimeout {
	return &FunctionWithTimeout{
		f,
		timeout,
	}
}

// WithDynamicContext returns a simulated function that will be run with dynamic context timeout
func (f Function) WithDynamicContext(weight float64, isPriority bool) *FunctionWithDynamiContext {
	return &FunctionWithDynamiContext{
		f,
		weight,
		isPriority,
	}
}

// FunctionWithTimeout denotes a function simulation with context timeout
type FunctionWithTimeout struct {
	Function
	timeout int
}

// Run runs the function
func (f *FunctionWithTimeout) Run(ctx context.Context, w io.Writer) {
	time.Sleep(time.Duration(f.timeout) * time.Millisecond)
	f.isExecuted = true
	fmt.Fprintf(w, rowFormat, f.name, f.timeout, getDeadline(ctx))
}

// IsExecuted returns true if function has been executed
func (f *FunctionWithTimeout) IsExecuted() bool {
	return f.isExecuted
}

func (f *FunctionWithTimeout) String() string {
	return f.name
}

// FunctionWithDynamiContext denotes a function simulation with dynamic context timeout
type FunctionWithDynamiContext struct {
	Function
	weight     float64
	isPriority bool
}

// Run runs the function
func (f *FunctionWithDynamiContext) Run(ctx context.Context, w io.Writer) {
	dynamicContext, esCancel := getNewContext(ctx, f.weight, f.isPriority)
	defer esCancel()
	timeout := getDeadline(dynamicContext)
	time.Sleep(time.Duration(timeout) * time.Millisecond)
	f.isExecuted = true
	fmt.Fprintf(w, rowFormat, f.name, timeout, getDeadline(ctx))
}

// IsExecuted returns true if function has been executed
func (f *FunctionWithDynamiContext) IsExecuted() bool {
	return f.isExecuted
}

func (f *FunctionWithDynamiContext) String() string {
	return f.name
}

// Simulator denotes a budgeting simulator
type Simulator struct {
	name    string
	timeout int
	process []Proccess
}

// NewSimulator returns new simulator
func NewSimulator(name string, timeout int) *Simulator {
	return &Simulator{
		name:    name,
		timeout: timeout,
	}
}

// RegisterFunctions set process need to be simulated
func (s *Simulator) RegisterFunctions(ps ...Proccess) {
	s.process = ps
}

// Run start the simulator
func (s *Simulator) Run() {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.Debug)
	fmt.Fprint(w, "=====================\n")
	fmt.Fprintf(w, "SIMULATOR:%s\n", s.name)
	fmt.Fprint(w, "Name\tMax Timeout(ms)\tRemaining(ms)\t\n")
	fmt.Fprintf(w, rowFormat, "Init", s.timeout, s.timeout)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.timeout)*time.Millisecond)
	defer cancel()

	done := make(chan int64, 1)

	go func() {
		for _, p := range s.process {
			p.Run(ctx, w)
		}
		done <- getDeadline(ctx)
	}()

	select {
	case <-ctx.Done():
		fmt.Fprint(w, "Time out reached with unexecuted function: \n")
		for _, p := range s.process {
			if !p.IsExecuted() {
				fmt.Fprintf(w, "- %s\n", p.String())
			}
		}
	case timeLeft := <-done:
		fmt.Fprintf(w, "Done with time left %v ms\n", timeLeft)
	}
	fmt.Fprint(w, "=====================\n")
	w.Flush()
}

func getDeadline(ctx context.Context) int64 {
	deadline, _ := ctx.Deadline()

	unixTime := deadline.UnixNano()
	diffTime := unixTime - time.Now().UnixNano()
	diffTime = diffTime / 1e6

	return diffTime
}

func getNewContext(ctx context.Context, percentage float64, isPriority bool) (context.Context, context.CancelFunc) {
	timeout := getDeadline(ctx)
	timeoutThreshold := 30

	newTimeout := float64(timeout) * percentage
	if newTimeout < float64(timeoutThreshold) && isPriority == true {
		newTimeout = float64(timeout)
	}

	newCtx, cancel := context.WithTimeout(ctx, time.Duration(newTimeout)*time.Millisecond)

	return newCtx, cancel
}
