// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in LICENSE.gorilla.

/*
Package lazycookie encodes and decodes authenticated, and optionally
encrypted, cookie values.

Use this package directly when an application needs to put a small trusted
value in an HTTP cookie without using the GoLazy session layer. GoLazy sessions
normally reach this package through lazysession.CookieStore, which builds
lazycookie codecs and calls EncodeMulti and DecodeMulti. lazyapp configures
that path when lazyapp.Config.Sessions is enabled, so most GoLazy apps should
configure lazysession instead of constructing SecureCookie values themselves.

A secure cookie is authenticated with HMAC. Authentication means the browser
can still send the value back, but it cannot change the value, timestamp, or
cookie name without making Decode fail. HMAC does not hide the content. Pass a
block key to New when the cookie content must also be encrypted. The default
block cipher is AES, used in CTR mode with a fresh initialization vector for
each encoded value.

To use lazycookie directly, first create a SecureCookie:

	var hashKey = []byte("32-byte-authentication-secret!!")
	var blockKey = []byte("16-byte-aes-key!!")
	var s = lazycookie.New(hashKey, blockKey)

The hashKey is required and should be 32 or 64 bytes. The blockKey is optional;
set it to nil to sign without encrypting. When blockKey is set for the default
AES cipher, it must be 16, 24, or 32 bytes to select AES-128, AES-192, or
AES-256.

GenerateRandomKey can create strong keys, but it does not persist them. Store
generated keys in configuration or a secret manager. If an application creates
new keys every time it starts, cookies issued before the restart cannot be
decoded.

Encode serializes the value, optionally encrypts it, adds a timestamp, signs
the cookie name, timestamp, and value, and returns a base64url cookie value:

	func SetCookieHandler(w http.ResponseWriter, r *http.Request) {
		value := map[string]string{"foo": "bar"}
		encoded, err := s.Encode("cookie-name", value)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:  "cookie-name",
			Value: encoded,
			Path:  "/",
		})
	}

Decode verifies the signature and age before decrypting and deserializing. The
name passed to Decode must match the cookie name passed to Encode:

	func ReadCookieHandler(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("cookie-name")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		value := make(map[string]string)
		if err := s.Decode("cookie-name", cookie.Value, &value); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		fmt.Fprintf(w, "The value of foo is %q", value["foo"])
	}

The default serializer is encoding/gob. Basic values work without setup; custom
types stored behind interface values may need gob.Register. SetSerializer
switches to another Serializer, such as JSONEncoder or NopEncoder.

For key rotation, create multiple codecs with CodecsFromPairs. EncodeMulti
writes new cookies with the first working codec. DecodeMulti tries each codec
in order, which lets old cookies remain readable while new cookies use the new
keys.
*/
package lazycookie
