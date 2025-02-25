package responder

import (
	"fmt"
	"precisiondosing-api-go/cfg"
	"time"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

type Mailer struct {
	mailInfo  cfg.MailerConfig
	metaInfo  cfg.MetaConfig
	debugMode bool
}

func NewMailer(mailerSettings cfg.MailerConfig, metaInfo cfg.MetaConfig, debug bool) *Mailer {
	return &Mailer{
		mailInfo:  mailerSettings,
		metaInfo:  metaInfo,
		debugMode: debug,
	}
}

func (m *Mailer) SendNewAccoundEmail(receiverName, receiver, token string, expirationTime time.Time) error {
	apiDomain := m.metaInfo.URL + m.metaInfo.Group
	body := newAccountMsg(
		receiverName,
		receiver,
		m.metaInfo.Name,
		apiDomain,
		token,
		expirationTime,
	)

	subject := fmt.Sprintf("New account created for %s", m.metaInfo.Name)
	err := m.send(subject, body, receiverName, receiver)
	if err != nil {
		return fmt.Errorf("cannot send email for new account: %w", err)
	}

	return nil
}

func (m *Mailer) SendPasswordResetEmail(receiverName, receiver, token string, expirationTime time.Time) error {
	apiDomain := m.metaInfo.URL + m.metaInfo.Group
	body := passwordResetMsg(
		receiver,
		m.metaInfo.Name,
		apiDomain,
		token,
		expirationTime,
	)

	subject := fmt.Sprintf("Password reset requested for %s", m.metaInfo.Name)
	err := m.send(subject, body, receiverName, receiver)
	if err != nil {
		return fmt.Errorf("cannot send email for password reset: %w", err)
	}

	return nil
}

func (m *Mailer) SendChangeEmail(receiverName, receiver, token string, expirationTime time.Time) error {
	apiDomain := m.metaInfo.URL + m.metaInfo.Group
	body := changeEmailMsg(
		m.metaInfo.Name,
		apiDomain,
		token,
		expirationTime,
	)

	subject := fmt.Sprintf("Email change requested for %s", m.metaInfo.Name)
	err := m.send(subject, body, receiverName, receiver)
	if err != nil {
		return fmt.Errorf("cannot send email for email change: %w", err)
	}

	return nil
}

func changeEmailMsg(apiName, apiDomain, token string, expirationTime time.Time) string {
	return fmt.Sprintf(
		`Dear user,

A change of email login for %s was requested.
If you did not request this, please ignore this email and your email will remain unchanged.

To confirm the change, use this endpoint:

https://%s/user/email/confirm

with token: %s

As an example, you can use the following curl command:

curl -X POST https://%s/user/email/confirm \
-H "Authorization: Bearer <your_jwt_token>" \
-H "Content-Type: application/json" \
-d '{
	"token": "%s",
}'

You have to be authenticated with your previously registered email.

Swagger documentation for the API can be found at %s/swagger

This token will expire at %s. If you need a new token, please contact the administrator.`,
		apiName, apiDomain, token, apiDomain, token, apiDomain, expirationTime.Format(time.RFC1123))
}

func newAccountMsg(fullName, email, apiName, apiDomain, token string, expirationTime time.Time) string {
	return fmt.Sprintf(
		`Dear %s,

An account for the %s was created.
To get started, please set your initial password by using this endpoint:

https://%s/user/password/init

with token: %s

As an example, you can use the following curl command:

curl -X POST https://%s/user/password/init \
-H "Content-Type: application/json" \
-d '{
	"token": "%s",
	"email": "%s",
	"password": "your_secure_password"
}'

Swagger documentation for the API can be found at %s/swagger

This token will expire at %s. If you need a new token, please contact the administrator.`,
		fullName, apiName, apiDomain, token, apiDomain, token, email, apiDomain, expirationTime.Format(time.RFC1123))
}

func passwordResetMsg(email, apiName, apiDomain, token string, expirationTime time.Time) string {
	return fmt.Sprintf(
		`Dear user,

An password reset for %s was requested.
If you did not request this, please ignore this email and your password will remain unchanged.
To reset your password, use this endpoint:

https://%s/user/password/reset/confirm

with token: %s

As an example, you can use the following curl command:

curl -X POST https://%s/user/password/reset/confirm \
-H "Content-Type: application/json" \
-d '{
	"token": "%s",
	"email": "%s",
	"password": "your_new_secure_password"
}'

Swagger documentation for the API can be found at %s/swagger

This token will expire at %s.`,
		apiName, apiDomain, token, apiDomain, token, email, apiDomain, expirationTime.Format(time.RFC1123))
}

func (m *Mailer) send(subject, body, receiverName, receiver string) error {
	from := mail.NewEmail(m.metaInfo.Name, m.mailInfo.SendEmail)
	var toMail string
	if m.debugMode {
		toMail = m.mailInfo.SendEmail
	} else {
		toMail = receiver
	}
	to := mail.NewEmail(receiverName, toMail)

	message := mail.NewSingleEmailPlainText(from, subject, to, body)
	client := sendgrid.NewSendClient(m.mailInfo.APIKey)
	_, err := client.Send(message)
	if err != nil {
		return err //nolint:wrapcheck // no need to wrap since this an internal function
	}

	return nil
}
