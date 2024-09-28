package closer

import (
	"errors"
	"log"
	"os"
	"os/signal"
	"sync"
)

type Closer struct {
	mu        sync.Mutex
	once      sync.Once
	done      chan struct{}
	functions []func() error
}

func NewCloser(sig ...os.Signal) *Closer {
	c := &Closer{done: make(chan struct{})}
	if len(sig) > 0 {
		go func() {
			ch := make(chan os.Signal, 1)
			signal.Notify(ch, sig...)
			<-ch
			signal.Stop(ch)
			if err := c.Close(); err != nil {
				log.Print(err)
			}
		}()
	}
	return c
}

func (c *Closer) Add(f ...func() error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.functions = append(c.functions, f...)
}

func (c *Closer) Wait() {
	<-c.done
}

func (c *Closer) Close() error {
	c.mu.Lock()
	functions := c.functions
	c.mu.Unlock()

	n := len(functions)
	errs := make([]error, n)

	c.once.Do(func() {
		defer close(c.done)

		wg := sync.WaitGroup{}
		wg.Add(n)

		for i, f := range functions {
			go func() {
				defer wg.Done()
				errs[i] = f()
			}()
		}

		wg.Wait()
	})

	return errors.Join(errs...)
}
