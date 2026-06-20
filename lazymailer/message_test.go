package lazymailer

import (
	"net/mail"
	"strings"
	"testing"
)

func TestMessageBytesBuildsMultipartAlternative(t *testing.T) {
	message := Message{
		From:    mail.Address{Name: "Sender", Address: "sender@example.com"},
		To:      []mail.Address{{Address: "ada@example.com"}},
		Subject: "Hello",
		Text:    "Plain",
		HTML:    "<strong>HTML</strong>",
	}

	raw, err := message.Bytes()
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	body := string(raw)
	for _, expected := range []string{
		"From: \"Sender\" <sender@example.com>\r\n",
		"To: <ada@example.com>\r\n",
		"Subject: Hello\r\n",
		"Content-Type: multipart/alternative;",
		"Content-Type: text/plain; charset=utf-8",
		"Content-Type: text/html; charset=utf-8",
		"Plain",
		"<strong>HTML</strong>",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("message does not contain %q:\n%s", expected, body)
		}
	}
}
