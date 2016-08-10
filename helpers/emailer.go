package helpers

import (
	"html/template"
	"io"

	"github.com/satori/go.uuid"
	gomail "gopkg.in/gomail.v2"
)

type EmailerOpts struct {
	Host     string
	Port     int
	User     string
	Password string
	Secure   bool

	From         string
	Subject      string
	Domain       string // used for the Message-Id header
	TextTemplate *template.Template
	HtmlTemplate *template.Template
}

// Abstracts efficient email sending.
// It isn't thread safe.
type Emailer struct {
	opt EmailerOpts

	msg    *gomail.Message
	data   *map[string]string
	dialer *gomail.Dialer
}

func NewEmailer(opt EmailerOpts) (*Emailer, error) {
	e := new(Emailer)
	e.opt = opt

	// Prepare message
	e.msg = gomail.NewMessage()
	e.msg.SetHeader("From", opt.From)
	e.msg.SetHeader("Subject", opt.Subject)
	if e.opt.TextTemplate != nil {
		e.msg.AddAlternativeWriter("text/plain", func(writer io.Writer) error {
			return e.opt.TextTemplate.Execute(writer, *e.data)
		})
	}

	if e.opt.HtmlTemplate != nil {
		e.msg.AddAlternativeWriter("text/html", func(writer io.Writer) error {
			return e.opt.HtmlTemplate.Execute(writer, *e.data)
		})
	}

	// Dialer will send emails
	e.dialer = &gomail.Dialer{
		Host:     opt.Host,
		Port:     opt.Port,
		Username: opt.User,
		Password: opt.Password,
		SSL:      opt.Secure,
		// LocalName: opt.Domain,
	}

	return e, nil
}

func (e *Emailer) Send(to string, args map[string]string) error {
	// Set message-id to avoid spam filters
	// <[uid]@[sendingdomain.com]>
	id := uuid.NewV4()
	e.msg.SetHeader("Message-Id", "<"+id.String()+"@"+e.opt.Domain+">")

	e.data = &args
	e.msg.SetHeader("To", to)

	err := e.dialer.DialAndSend(e.msg)
	return err
}
