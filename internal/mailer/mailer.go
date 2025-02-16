package mailer

type Mailer interface {
	Send(recipient, templateFile string, data any) error
}
