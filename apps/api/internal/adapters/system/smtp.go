package system

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/smtp"
	"os"
	"time"

	"github.com/mattwebhub/micro1-template/apps/api/internal/domain"
)

const (
	SMTPTLSNone     = "none"
	SMTPTLSStartTLS = "starttls"
	SMTPTLSImplicit = "implicit"
)

var errSMTPDelivery = errors.New("smtp: verification delivery failed")

type SMTPMailerConfig struct {
	Address       string
	Username      string
	Password      string
	From          string
	TLSMode       string
	TLSServerName string
	TLSCAFile     string
}

type SMTPMailer struct {
	config    SMTPMailerConfig
	tlsConfig *tls.Config
}

func NewSMTPMailer(config SMTPMailerConfig) (*SMTPMailer, error) {
	host, _, err := net.SplitHostPort(config.Address)
	if err != nil || host == "" || config.From == "" {
		return nil, errors.New("smtp: address and sender are required")
	}
	if (config.Username == "") != (config.Password == "") {
		return nil, errors.New("smtp: username and password must be configured together")
	}

	mailer := &SMTPMailer{config: config}
	switch config.TLSMode {
	case SMTPTLSNone:
		return mailer, nil
	case SMTPTLSStartTLS, SMTPTLSImplicit:
		if config.TLSServerName == "" {
			return nil, errors.New("smtp: TLS server name is required")
		}
	default:
		return nil, errors.New("smtp: invalid TLS mode")
	}

	rootCAs, err := smtpRootCAs(config.TLSCAFile)
	if err != nil {
		return nil, err
	}
	mailer.tlsConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
		ServerName: config.TLSServerName,
		RootCAs:    rootCAs,
	}
	return mailer, nil
}

func smtpRootCAs(caFile string) (*x509.CertPool, error) {
	if caFile == "" {
		return nil, nil
	}
	rootCAs, err := x509.SystemCertPool()
	if err != nil || rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}
	pemBytes, err := os.ReadFile(caFile)
	if err != nil || !rootCAs.AppendCertsFromPEM(pemBytes) {
		return nil, errors.New("smtp: TLS CA file is invalid")
	}
	return rootCAs, nil
}

func (mailer *SMTPMailer) SendVerification(ctx context.Context, email string, _ domain.Audience, code string, expires time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	connection, err := (&net.Dialer{}).DialContext(ctx, "tcp", mailer.config.Address)
	if err != nil {
		return smtpDeliveryError(ctx)
	}
	defer connection.Close()
	if deadline, ok := ctx.Deadline(); ok {
		if err := connection.SetDeadline(deadline); err != nil {
			return smtpDeliveryError(ctx)
		}
	}
	stopCancellation := context.AfterFunc(ctx, func() {
		_ = connection.SetDeadline(time.Now())
	})
	defer stopCancellation()

	client, err := mailer.smtpClient(ctx, connection)
	if err != nil {
		return smtpDeliveryError(ctx)
	}
	defer client.Close()

	if mailer.config.Username != "" {
		if err := client.Auth(smtpPlainAuth{username: mailer.config.Username, password: mailer.config.Password}); err != nil {
			return smtpDeliveryError(ctx)
		}
	}
	if err := client.Mail(mailer.config.From); err != nil {
		return smtpDeliveryError(ctx)
	}
	if err := client.Rcpt(email); err != nil {
		return smtpDeliveryError(ctx)
	}
	writer, err := client.Data()
	if err != nil {
		return smtpDeliveryError(ctx)
	}
	message := []byte("From: " + mailer.config.From + "\r\nTo: " + email + "\r\nSubject: ClaimBounty verification code\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nYour ClaimBounty code is " + code + ". It expires at " + expires.UTC().Format(time.RFC3339) + ".\r\n")
	if _, err := writer.Write(message); err != nil {
		_ = writer.Close()
		return smtpDeliveryError(ctx)
	}
	if err := writer.Close(); err != nil {
		return smtpDeliveryError(ctx)
	}
	if err := client.Quit(); err != nil {
		return smtpDeliveryError(ctx)
	}
	return nil
}

func (mailer *SMTPMailer) smtpClient(ctx context.Context, connection net.Conn) (*smtp.Client, error) {
	host, _, err := net.SplitHostPort(mailer.config.Address)
	if err != nil {
		return nil, err
	}
	switch mailer.config.TLSMode {
	case SMTPTLSNone:
		return smtp.NewClient(connection, host)
	case SMTPTLSImplicit:
		tlsConnection := tls.Client(connection, mailer.tlsConfig.Clone())
		if err := tlsConnection.HandshakeContext(ctx); err != nil {
			return nil, err
		}
		return smtp.NewClient(tlsConnection, host)
	case SMTPTLSStartTLS:
		client, err := smtp.NewClient(connection, host)
		if err != nil {
			return nil, err
		}
		if supported, _ := client.Extension("STARTTLS"); !supported {
			_ = client.Close()
			return nil, errors.New("smtp: STARTTLS is required")
		}
		if err := client.StartTLS(mailer.tlsConfig.Clone()); err != nil {
			_ = client.Close()
			return nil, err
		}
		return client, nil
	default:
		return nil, errors.New("smtp: invalid TLS mode")
	}
}

func smtpDeliveryError(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return errSMTPDelivery
}

// smtpPlainAuth is used after this adapter has established its configured
// transport. net/smtp's PlainAuth cannot identify an implicit TLS connection.
type smtpPlainAuth struct {
	username string
	password string
}

func (auth smtpPlainAuth) Start(*smtp.ServerInfo) (string, []byte, error) {
	return "PLAIN", []byte("\x00" + auth.username + "\x00" + auth.password), nil
}

func (smtpPlainAuth) Next([]byte, bool) ([]byte, error) {
	return nil, errors.New("smtp: unexpected authentication challenge")
}
