package express

import (
	"bytes"
	"fmt"
	"html/template"
	"net/mail"
	"net/smtp"
	"path"
	"strings"
)

type MailerConfig struct {
	Host          string
	Port          int
	Username      string
	Password      string
	From          *mail.Address
	TemplatesPath string
}

func encodeRFC2047(String string) string {
	addr := mail.Address{String, ""}
	return strings.Trim(addr.String(), " <>")
}

type MailerTemplate struct {
	Files   []string
	To      mail.Address
	Subject string
	Data    interface{}
}

type Mailer interface {
	Parse(*MailerTemplate) (string, error)
	Send(*MailerTemplate) (error)
}

type SimpleMailer struct {
	*MailerConfig
}

func (e *SimpleMailer) Parse(et *MailerTemplate) (body string, err error) {

	var doc bytes.Buffer

	ts := make([]string, 0)

	for _, v := range et.Files {
		ts = append(ts, path.Join(e.TemplatesPath, v))
	}

	t := template.Must(template.ParseFiles(ts...))

	err = t.Execute(&doc, struct {
		From    string
		To      string
		Subject string
		Data    interface{}
	}{
		e.From.String(),
		et.To.String(),
		et.Subject,
		et.Data,
	})

	if err != nil {
		return "", err
	}

	return doc.String(), nil
}

func (e *SimpleMailer) Send(et *MailerTemplate) (err error) {

	body, err := e.Parse(et)
	if err != nil {
		return err
	}

	server := fmt.Sprintf("%s:%d", e.Host, e.Port)

	auth := smtp.PlainAuth(
		"",
		e.Username,
		e.Password,
		e.Host,
	)

	go smtp.SendMail(server, auth, e.From.Address, []string{et.To.Address}, []byte(body))

	return nil

}

func NewSimpleMailer(config *MailerConfig) Mailer {
	return &SimpleMailer{config}
}