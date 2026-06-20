package lazymailer

import (
	"bytes"
	"fmt"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"net/textproto"
	"strings"
)

type Message struct {
	From    mail.Address
	To      []mail.Address
	Subject string
	Headers textproto.MIMEHeader
	Text    string
	HTML    string
}

type Address = mail.Address

func (m Message) Bytes() ([]byte, error) {
	if strings.TrimSpace(m.From.Address) == "" {
		return nil, fmt.Errorf("lazymailer: from address is required")
	}
	if len(m.To) == 0 {
		return nil, fmt.Errorf("lazymailer: at least one recipient is required")
	}

	var out bytes.Buffer
	headers := cloneHeader(m.Headers)
	headers.Set("From", m.From.String())
	headers.Set("To", addressList(m.To))
	headers.Set("Subject", mime.QEncoding.Encode("utf-8", m.Subject))
	headers.Set("MIME-Version", "1.0")

	switch {
	case m.Text != "" && m.HTML != "":
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		headers.Set("Content-Type", mime.FormatMediaType("multipart/alternative", map[string]string{
			"boundary": writer.Boundary(),
		}))
		if err := writePart(writer, "text/plain; charset=utf-8", m.Text); err != nil {
			return nil, err
		}
		if err := writePart(writer, "text/html; charset=utf-8", m.HTML); err != nil {
			return nil, err
		}
		if err := writer.Close(); err != nil {
			return nil, err
		}
		writeHeaders(&out, headers)
		out.WriteString("\r\n")
		out.Write(body.Bytes())
	case m.HTML != "":
		headers.Set("Content-Type", "text/html; charset=utf-8")
		headers.Set("Content-Transfer-Encoding", "quoted-printable")
		writeHeaders(&out, headers)
		out.WriteString("\r\n")
		writeQuotedPrintable(&out, m.HTML)
	default:
		headers.Set("Content-Type", "text/plain; charset=utf-8")
		headers.Set("Content-Transfer-Encoding", "quoted-printable")
		writeHeaders(&out, headers)
		out.WriteString("\r\n")
		writeQuotedPrintable(&out, m.Text)
	}

	return out.Bytes(), nil
}

func Recipients(addresses []mail.Address) []string {
	recipients := make([]string, 0, len(addresses))
	for _, address := range addresses {
		recipients = append(recipients, address.Address)
	}
	return recipients
}

func ParseAddress(value string) (mail.Address, error) {
	address, err := mail.ParseAddress(value)
	if err != nil {
		return mail.Address{}, err
	}
	return *address, nil
}

func MustParseAddress(value string) mail.Address {
	address, err := ParseAddress(value)
	if err != nil {
		panic(err)
	}
	return address
}

func cloneHeader(source textproto.MIMEHeader) textproto.MIMEHeader {
	out := textproto.MIMEHeader{}
	for key, values := range source {
		for _, value := range values {
			out.Add(key, value)
		}
	}
	return out
}

func addressList(addresses []mail.Address) string {
	parts := make([]string, 0, len(addresses))
	for _, address := range addresses {
		parts = append(parts, address.String())
	}
	return strings.Join(parts, ", ")
}

func writeHeaders(out *bytes.Buffer, headers textproto.MIMEHeader) {
	for key, values := range headers {
		for _, value := range values {
			out.WriteString(key)
			out.WriteString(": ")
			out.WriteString(value)
			out.WriteString("\r\n")
		}
	}
}

func writePart(writer *multipart.Writer, contentType string, body string) error {
	header := textproto.MIMEHeader{}
	header.Set("Content-Type", contentType)
	header.Set("Content-Transfer-Encoding", "quoted-printable")
	part, err := writer.CreatePart(header)
	if err != nil {
		return err
	}
	writeQuotedPrintable(part, body)
	return nil
}

func writeQuotedPrintable(out interface{ Write([]byte) (int, error) }, body string) {
	writer := quotedprintable.NewWriter(out)
	_, _ = writer.Write([]byte(body))
	_ = writer.Close()
}
