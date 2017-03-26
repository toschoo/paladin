// Package paladin provides simple protection of critical resources against
// asynchronous interruption signals sent by the operating systems.
// Paladin provides a Run method that expects 
// 
// - a function to obtain a resource (which must be a Closer)
//
// - a function to release the resource (using Close)
//
// - and a function that is run in between  
// obtaining and releasing the resource; the user application
// should entirely live within this function.
//
// Currently, only SIGINT is handled and the behaviour is to
// close the program.
// More sophisticated behaviour and more signals will be provided
// in the future.
package paladin

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
)

// Opener obtains the resources
type Opener func() (io.Closer, error)

// Closer releases the resources
type Closer func(io.Closer) error

// Runner is the home of the user application.
// The interface passed in is the resource
// obtained by the Opener.
type Runner func(interface{})

// Paladin implements the signal handler
// and protection for applications.
type Paladin struct {
	// Signal stores the signal that occurred.
	// If no signal has occurred, Signal is nil.
	Signal os.Signal
	guard sync.Mutex
	// signal list
}

// add signals

// New creates a new Paladin
func New() *Paladin {
	p := new(Paladin)
	p.Signal = nil
	p.Enter()
	return p
}

// Enter protects critical sections that must be finished
// before the program terminates.
// A typical use case is transactions.
// Suppose the user code needs to write a set of records
// into a file and it needs to write them either completely
// or not at all. This sequendce of write operations
// should be protected:
//
//     p.Enter()
//     operation1()
//     operation2()
//     ...
//     p.Leave()
//
// It is usually not necessary to protect single write operations.
// Paladin will always release the resources before terminating
// the program. Well implemented resource interfaces will 
// guarantee that the resource is in a clean state after closing.
func (p *Paladin) Enter() {
	p.guard.Lock()
}

// Leave signalsl the end of the critical section to the paladin.
func (p *Paladin) Leave() {
	p.guard.Unlock()
}

// event is either an operating system signal
// or an internal event (i.e.: user application has terminated)
type event struct {
	os bool
	s  os.Signal
}

// Run receives an Opener, a Closer and a Runner.
// It will first set up a signal handler;
// then it will obtain the critical resource by calling the Opener;
// then it will start the Runner (in its own goroutine) and wait
// until either the Runner terminates or an iterruption occurs.
// It either case, it will close the critical resource 
// by calling the Closer on it.
// If a signal has occurred, it is stored 
// in the Signal field of the Paladin.
func (p *Paladin) Run(openr Opener, closr Closer, run Runner) (err error) {
	err = nil

	// install signal handler
	sig := make(chan os.Signal, 1024) // can we live with a smaller channel?
	signal.Notify(sig, os.Interrupt)

	// set up internal event queue
	done := make(chan event)

	// Wait for signals
	go func() {
		var e event
		e.os = true
		e.s  = <-sig
		done <- e
	}()

	// Obtain resources
	c, err := openr()
	if err != nil {
		msg := fmt.Sprintf("Could not open: %v", err)
		return errors.New(msg)
	}

	// Allow runner to enter critical code
	p.Leave()

	// Runner
	go func() {
		run(c)
		var e event
		e.os = false
		done <- e
	}()

	// wait for events
	e := <-done

	// Block runner from entering critical code
	p.Enter()

	if e.os {
		p.Signal = e.s
	}

	// Close resource
	err = closr(c)
	return
}
