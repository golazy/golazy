// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in LICENSE.gorilla.

/*
Package lazysession stores per-browser request state in signed cookies or in a
custom session store.

A session is a named map of values associated with an HTTP request. Stores load
that map from the incoming request and save it to the outgoing response.
CookieStore keeps the whole session value inside one authenticated cookie.
FilesystemStore keeps only a signed session ID in the cookie and stores the
values in files. Application-specific backends can implement Store when the
values should live somewhere else, such as a database.

CookieStore and FilesystemStore use lazycookie for signing, decoding, optional
encryption, age checks, and key rotation. Authentication means a browser can
send a cookie back but cannot alter its value or cookie name without making
decode fail. Authentication does not hide the content; pass authentication and
block keys as pairs when cookie contents must also be encrypted. The first key
pair writes new cookies. Later pairs are read-only fallbacks for old cookies.

The smallest standalone use creates a store, gets a named session, changes
Values, and saves before writing the response:

	var store = lazysession.NewCookieStore([]byte("32-byte-authentication-secret!!"))

	func handler(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, "app_session")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		session.Values["user_id"] = "42"
		if err := session.Save(r, w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, "saved")
	}

NewCookieStore expects stable secrets. Generate strong keys with
lazycookie.GenerateRandomKey or a secret manager, then persist them in
configuration. If a process creates new keys on every startup, existing session
cookies cannot be decoded after a restart.

Manager is the GoLazy application layer over Store. A Manager owns the default
session name and store for one app. Its Handler installs the manager in the
request context, creates the per-request registry, runs the next handler, and
saves all sessions registered during the request before the first response
write. This is the layer used by lazyapp.

Enable sessions in a GoLazy app through lazyapp.Config.Sessions:

	app := lazyapp.New(lazyapp.Config{
		Name: "shop",
		Sessions: lazysession.Config{
			Key: os.Getenv("SESSION_KEY"),
			Options: &lazysession.Options{
				Path:     "/",
				MaxAge:   86400 * 30,
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			},
		},
	})

lazyapp calls NewManager, stores the manager in the app context with
WithManager, and installs Manager.Handler in the app middleware stack. When the
session name is empty, lazyapp derives it from Config.Name and appends
"_session"; without lazyapp the package default is "lazy_session". Config.Key is
expanded with SHA-256 before the cookie store is created. Use Config.KeyPairs
for explicit lazycookie key pairs or Config.Store to provide a custom Store.

Handlers that run under Manager.Handler can use Get to retrieve the configured
application session:

	func handler(w http.ResponseWriter, r *http.Request) {
		session, err := lazysession.Get(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		session.Values["flash"] = "welcome"
	}

Get reads the Manager from r.Context. ManagerFromContext and WithManager are
available for code that builds its own server stack or wants to pass the
manager through a base context. Store.Get remains useful when one request needs
multiple named sessions or a package intentionally stays independent of the
GoLazy app manager.

The request Registry is the reason Save(r, w) can persist every session touched
during a request. Store.Get registers the decoded session by name; Save walks
that registry and calls the matching Store.Save for each session. Manager.Handler
does this automatically. If a handler uses stores directly without the manager,
call Session.Save for one session or Save(r, w) for all registered sessions
before writing the response.

Flash messages are values that last until read. AddFlash stores a value under
the default "_flash" key, or under a custom key when one is passed. Flashes
returns the stored values and removes them from the session, which makes them
useful for messages shown after a redirect.

Session values are encoded with encoding/gob by default. Basic values work
without setup. Custom values stored through interface slots, including flash
values, may need gob.Register during program initialization:

	type Notice struct {
		Kind string
		Text string
	}

	func init() {
		gob.Register(Notice{})
	}

Options are copied from the store to each new session. Changing store.Options
affects future sessions; changing session.Options affects only that response.
MaxAge follows http.Cookie semantics: zero omits Max-Age and expires at browser
shutdown, a negative value deletes the cookie immediately, and a positive value
sets a lifetime in seconds.
*/
package lazysession
