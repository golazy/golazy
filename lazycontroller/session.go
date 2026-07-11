package lazycontroller

import (
	"fmt"
	"strings"

	"golazy.dev/lazysession"
)

// SessionGet reads a value from the application session without marking it for save.
func (b *Base) SessionGet(key string) (any, bool, error) {
	session, err := b.sessionForRead()
	if err != nil {
		return nil, false, err
	}
	value, ok := session.Values[key]
	return value, ok, nil
}

// SessionSet writes a value to the application session and marks it for save.
func (b *Base) SessionSet(key string, value any) error {
	session, err := b.sessionForRead()
	if err != nil {
		return err
	}
	session.Values[key] = value
	return b.markSessionDirty()
}

// SessionDelete removes a value from the application session when it exists.
func (b *Base) SessionDelete(key string) error {
	session, err := b.sessionForRead()
	if err != nil {
		return err
	}
	if _, ok := session.Values[key]; !ok {
		return nil
	}
	delete(session.Values, key)
	return b.markSessionDirty()
}

// FlashSet adds flash values under level and marks the session for save.
func (b *Base) FlashSet(level string, values ...any) error {
	level = strings.TrimSpace(level)
	if level == "" {
		return fmt.Errorf("lazycontroller: flash level is required")
	}
	session, err := b.sessionForRead()
	if err != nil {
		return err
	}
	for _, value := range values {
		session.AddFlash(value, level)
	}
	return b.markSessionDirty()
}

// FlashGet returns and consumes flash values under level.
func (b *Base) FlashGet(level string) ([]any, error) {
	level = strings.TrimSpace(level)
	if level == "" {
		return nil, fmt.Errorf("lazycontroller: flash level is required")
	}
	session, err := b.sessionForRead()
	if err != nil {
		return nil, err
	}
	if _, ok := session.Values[level]; !ok {
		return nil, nil
	}
	values := session.Flashes(level)
	if err := b.markSessionDirty(); err != nil {
		return nil, err
	}
	return values, nil
}

func (b *Base) sessionForRead() (*lazysession.Session, error) {
	if b == nil || b.request == nil {
		return nil, fmt.Errorf("lazycontroller: controller request is not initialized")
	}
	if b.sessionSet {
		return b.session, nil
	}
	session, err := lazysession.Read(b.request)
	if err != nil {
		return nil, err
	}
	b.session = session
	b.sessionSet = true
	return session, nil
}

func (b *Base) markSessionDirty() error {
	if b == nil || b.request == nil {
		return fmt.Errorf("lazycontroller: controller request is not initialized")
	}
	if !b.sessionSet || b.session == nil {
		return fmt.Errorf("lazycontroller: controller session is not initialized")
	}
	if b.sessionDirty {
		return nil
	}
	if err := lazysession.MarkDirty(b.request, b.session); err != nil {
		return err
	}
	b.sessionDirty = true
	return nil
}
