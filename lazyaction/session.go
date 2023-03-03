package lazyaction

import (
	"net/http"
	"strings"

	"github.com/gorilla/sessions"
)

type Session struct {
	s        *sessions.Session
	modified bool
}

var Store = sessions.NewCookieStore([]byte("//TODO: make this random and persistant"))

func (s *Session) loadFromRequest(r *http.Request) error {
	//gs gorilla session
	gs, err := Store.Get(r, "golazy_session")
	if err != nil {
		return err
	}
	s.s = gs
	return nil
}

func (s *Session) Get(key string) any {
	v := s.s.Values[key]
	return v
}

func (s *Session) Delete(key string) {
	if s.s.Values == nil {
		return
	}
	delete(s.s.Values, key)
	s.modified = true
}

func (s *Session) GetString(key string) string {
	v, ok := s.s.Values[key]
	if !ok || v == nil {
		return ""
	}

	str, ok := v.(string)
	if !ok {
		return ""
	}
	return str
}

func (s *Session) SetString(key, val string) {
	s.s.Values[key] = val
}

func (s *Session) Set(key string, val any) {
	s.modified = true
	s.s.Values[key] = val
}

func (s *Session) Flashes(vars ...string) []any {
	ret := s.s.Flashes(vars...)
	s.modified = true
	return ret
}

func (s *Session) AddFlash(val any, vars ...string) {
	s.s.AddFlash(val, vars...)
	s.modified = true
}

func (s *Session) SetFlash(key string, val string) {
	s.s.AddFlash(val, key)
	s.modified = true
}

func (s *Session) GetFlash(key string) string {
	flashes := s.s.Flashes(key)
	s.modified = true
	strs := []string{}
	for _, v := range flashes {
		if str, ok := v.(string); ok {
			strs = append(strs, str)
		}
	}
	return strings.Join(strs, ", ")
}

func (s *Session) SetError(err error) {
	s.SetFlash("error", err.Error())
}

func (s *Session) GetError() string {
	return s.GetFlash("error")
}

func (s *Session) SetNotice(notice string) {
	s.SetFlash("notice", notice)
}

func (s *Session) GetNotice() string {
	return s.GetFlash("notice")
}
