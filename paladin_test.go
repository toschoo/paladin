package paladin

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
)

const (
	iterations int = 100000
	numOfTests int = 100
)

type env struct {
	p       *Paladin
	result  int
	closed  bool
	stopper sync.Mutex
	killer  sync.Mutex
}

func (e *env) Close() error {
	e.closed = true
	return nil
}

// Tests that, when no signal occurs,
// - the paladin does not report a signal,
// - the user function always runs to the end and
// - the resource is closed
func TestNoInterrupt(t *testing.T) {
	e := new(env)
	p := New()
	opener := func() (Resource, error) {
		e.result = 0
		e.p = p
		e.closed = false
		return e, nil
	}
	closer := func(r Resource) error {
		e := r.(*env)
		return  e.Close()
	}
	p.Run(opener, closer, runner)
	if e.p.Signal != nil {
		t.Error(`Signal not nil without signal caught`)
	}
	if e.result != iterations {
		m := fmt.Sprintf("Program did not run through: %d", e.result)
		t.Error(m)
	}
	if !e.closed {
		t.Error(`Resource was not closed`)
	}
}

// Tests that, when one signal occurs,
// - either the paladin reports a signal or 
//   the function runs to the end and
// - the resource is closed
func TestOneInterrupt(t *testing.T) {
	for i:=0; i<numOfTests; i++ {
		// fmt.Printf("OneInterrupt: %d\n", i)
		err := testInterrupt(1)
		if err != nil {
			t.Error(err)
		}
	}
}

// Tests that, when n signals occur,
// - either the paladin reports a signal or 
//   the function runs to the end,
// - a protected block is either executed
//   completely or not at all
// - the resource is closed
func TestNInterrupts(t *testing.T) {
	for i:=0; i<numOfTests; i++ {
		testInterrupt(1000)
	}
}

// Tests that, when a signal occurs,
// - either the paladin reports a signal or 
//   the function runs to the end,
// - a protected block is either executed
//   completely or not at all
// - the resource is closed
func TestOneInterruptProtected(t *testing.T) {
	for i:=0; i<numOfTests; i++ {
		// fmt.Printf("OneInterruptProtected: %d\n", i)
		err := testInterruptProtected(1)
		if err != nil {
			t.Error(err)
		}
	}
}

// Tests that, when n signals occur,
// - either the paladin reports a signal or 
//   the function runs to the end,
// - a protected block is either executed
//   completely or not at all
// - the resource is closed
func TestNInterruptsProtected(t *testing.T) {
	for i:=0; i<numOfTests; i++ {
		// fmt.Printf("NInterruptProtected: %d\n", i)
		err := testInterruptProtected(1000)
		if err != nil {
			t.Error(err)
		}
	}
}

func testInterrupt(n int) error {
	p := New()
	e := new(env)
	opener := func() (Resource, error) {
		e.result = 0
		e.p = p
		e.closed = false
		e.stopper.Lock()
		e.killer.Lock()
		return e, nil
	}
	closer := func(r Resource) error {
		e := r.(*env)
		return  e.Close()
	}

	go killer(e,n)

	p.Run(opener, closer, func(r Resource) {
		e.killer.Unlock()
		e.stopper.Unlock()
		runner(r)
	})

	if !e.closed {
		return errors.New(`Resource was not closed`)
	}
	if p.Signal == nil && e.result != iterations {
		return errors.New(`Paladin could not catch signal`)
	}
	return nil
}

func testInterruptProtected(n int) error {
	p := New()
	e := new(env)
	opener := func() (Resource, error) {
		e.result = 0
		e.p = p
		e.closed = false
		e.stopper.Lock()
		e.killer.Lock()
		return e, nil
	}
	closer := func(r Resource) error {
		e := r.(*env)
		return  e.Close()
	}

	go killer(e,n)

	p.Run(opener, closer, func(r Resource) {
		e.stopper.Unlock()
		e.killer.Unlock()
		protectedRunner(r)
	})

	if !e.closed {
		return errors.New(`Resource was not closed`)
	}
	if p.Signal == nil && e.result <= iterations {
		return errors.New(`Paladin could not catch signal`)
	}
	if p.Signal != nil && !(e.result >= iterations || e.result == 0) {
		m := fmt.Sprintf("Program did not run through: %d", e.result)
		return errors.New(m)
	}
	return nil
}

func killer(e *env, n int) {
	myself, err := os.FindProcess(os.Getpid())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to find myself: %v", err)
		return
	}

	e.killer.Lock()
	defer e.killer.Unlock()

	for i:=0; i<n; i++ {
		// Not clear if this works on windows...
		err = myself.Signal(os.Interrupt)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to kill myself: %v", err)
			return
		}
	}
}

func runner(r Resource) {
	e := r.(*env)
	e.stopper.Lock()
	defer e.stopper.Unlock()
	for i:=0; i<iterations; i++ {
		e.result++
	}
}

func protectedRunner(r Resource) {
	e := r.(*env)

	e.stopper.Lock()
	defer e.stopper.Unlock()

	e.p.Enter()
	for i:=0; i<iterations; i++ {
		e.result++
	}
	e.p.Leave()
	e.result++
}
