package app

import (
	"os"
	"os/exec"
	"time"
)

type Cmd struct {
	exec.Cmd
	WaitTime time.Duration
	WaitCh   chan (struct{})
}

const DefaultWaitTime = time.Second

func (c *Cmd) Start() error {
	if c.WaitCh != nil {
		return nil
	}
	err := c.Cmd.Start()
	if err != nil {
		return err
	}
	c.WaitCh = make(chan (struct{}))
	go func() {
		c.Wait()
		close(c.WaitCh)
	}()
	return nil
}

func (c *Cmd) Stop() error {
	if c.WaitCh == nil {
		return nil
	}
	if err := c.Process.Signal(os.Interrupt); err != nil {
		return nil
	}
	wait := DefaultWaitTime
	if c.WaitTime != 0 {
		wait = c.WaitTime
	}
	select {
	case <-time.After(wait):
	case <-c.WaitCh:
		return nil
	}
	if err := c.Process.Signal(os.Kill); err != nil {
		return nil
	}
	<-c.WaitCh
	return nil
}
