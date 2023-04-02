package lazyaction

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gorilla/sessions"
)

var Store = sessions.NewCookieStore([]byte("//TODO: make this random and persistant"))

func init() {
	Store.Options = &sessions.Options{
		MaxAge: 60 * 60 * 24 * 30 * 6, // 6 months
		Secure: true,
		// Defaults to http.SameSiteDefaultMode
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
	}

}

type WithSession struct {
}

func (ws *WithSession) GenSession(w http.ResponseWriter, r *http.Request) (s *Session, err error) {
	s = &Session{
		w: w,
		r: r,
	}
	s.s, err = Store.Get(r, "golazy_session")
	return
}

type Session struct {
	s *sessions.Session
	w http.ResponseWriter
	r *http.Request
}

func (s *Session) Values() map[any]any {
	if s.s == nil {
		return make(map[any]any)
	}
	return s.s.Values

}
func (s *Session) GetAny(key string) any {
	v := s.s.Values[key]
	return v
}

func (s *Session) Delete(key string) {
	if s.s.Values == nil {
		return
	}
	delete(s.s.Values, key)
	s.s.Save(s.r, s.w)
}

func (s *Session) Get(key string) string {
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

func (s *Session) Set(key, val string) error {
	return s.SetAny(key, val)
}

func (s *Session) SetAny(key string, val any) error {
	s.s.Values[key] = val
	// save the session
	return s.s.Save(s.r, s.w)
}

func (s *Session) Store(key string, data any) error {
	j, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return s.Set(key, string(j))
}

func (s *Session) Load(key string, obj any) error {
	data := s.Get(key)
	if data == "" {
		return ErrNotFound
	}
	return json.Unmarshal([]byte(data), obj)
}

func (s *Session) Flashes(vars ...string) []any {
	ret := s.s.Flashes(vars...)
	return ret
}

func (s *Session) AddFlash(val any, vars ...string) {
	s.s.AddFlash(val, vars...)
}

func (s *Session) SetFlash(key string, val string) {
	s.s.AddFlash(val, key)
}

func (s *Session) GetFlash(key string) string {
	flashes := s.s.Flashes(key)
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
