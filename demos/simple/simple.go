// Simple Paladin demonstrator.
// The program expects a file path as command line parameter.
// It will open the file (Opener),
// read all its contents (so do not make it too big),
// and print it 10 times to stdout (Runner) with one second
// between the iterations.
// Finally, it will close the file (Closer),
// even when you interrupt the program with ^C.
//
// Usage example:
//   echo "hello world" > myfile.txt
//   ./simple myfile.txt
package main

import (
	"fmt"
	"github.com/toschoo/paladin"
	"io"
	"io/ioutil"
	"os"
	"time"
)

// There is only one Paladin, so it can be global.
var p *paladin.Paladin

// Do something with the resource
func say(s string) {
	p.Enter()
	defer p.Leave()
	fmt.Printf("%s",s)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("I need a file name\n")
		os.Exit(1)
	}
	nm := os.Args[1]
	opener := func() (io.Closer, error) {
		f, err := os.Open(nm)
		return f, err
	}
	closer := func(f io.Closer) error {
		err := f.Close()
		fmt.Printf("closed\n")
		return err
	}
	p = paladin.New()
	p.Run(opener, closer, func(r interface{}) {
		p.Enter()
		f := r.(io.Reader)
		b, err := ioutil.ReadAll(f)
		p.Leave()
		var s string
		if err != nil {
			s = fmt.Sprintf("ERROR: %v", err)
		} else {
			s = string(b)
		}
		for i:=0; i < 10; i++ {
			say(s)
			time.Sleep(time.Second)
		}
	})
	if p.Signal != nil {
		fmt.Printf("Signal occurred: %v\n", p.Signal)
	}
}
