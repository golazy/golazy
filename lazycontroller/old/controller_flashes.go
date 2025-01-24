package lazycontroller

import (
	"encoding/gob"
	"fmt"

	"golazy.dev/lazysupport"
)

type FlashMessager interface {
	Title() string
	Message() string
}

func (c *Controller) AddFlash(key string, value any) {
	c.sessionHasChanges = true
	var msg FlashMessage
	switch v := value.(type) {
	case FlashMessage:
		msg = v
	case *FlashMessage:
		msg = *v
	case FlashMessager:
		msg = FlashMessage{v.Title(), v.Message()}
	default:
		msg = FlashMessage{M: lazysupport.Truncate(fmt.Sprint(v), 50)}
	}
	fmt.Println(msg)
	fmt.Println(msg.Title())
	fmt.Println(msg.Message())
	c.session.AddFlash(msg, key)
}

func (c *Controller) AddNoticeFlash(msg any) {
	c.AddFlash(FlashNotice, msg)
}
func (c *Controller) AddWarningFlash(msg any) {
	c.AddFlash(FlashWarning, msg)
}
func (c *Controller) AddSuccessFlash(msg any) {
	c.AddFlash(FlashSuccess, msg)
}
func (c *Controller) AddErrorFlash(msg any) {
	c.AddFlash(FlashError, msg)
}

func (c *Controller) Flashes(key ...string) []FlashMessager {

	flashes := c.session.Flashes(key...)
	messages := make([]FlashMessager, len(flashes))
	for i, f := range flashes {
		if fm, ok := f.(FlashMessager); ok {
			messages[i] = fm
			continue
		}
		if s, ok := f.(string); ok {
			messages[i] = Flash(s)
			continue
		}
		messages[i] = Flash(fmt.Sprint(f))
	}
	if len(messages) > 0 {
		c.sessionHasChanges = true
	}
	return messages
}

type FlashMessage struct {
	T string
	M string
}

func init() {
	gob.Register(&Error{})
	gob.Register(&FlashMessage{})
}

func (f FlashMessage) Title() string {
	return f.T
}
func (f FlashMessage) Message() string {
	return f.M
}

type Flash string

func (s Flash) Title() string {
	return ""
}
func (s Flash) Message() string {
	return string(s)
}

func (c *Controller) FlashesByType() map[string][]any {
	m := map[string][]any{}
	for _, key := range []string{FlashNotice, FlashWarning, FlashSuccess, FlashError} {
		flashes := c.session.Flashes(key)
		if len(flashes) > 0 {
			m[key] = flashes
		}
	}
	if len(m) > 0 {
		c.sessionHasChanges = true
	}
	return m
}
