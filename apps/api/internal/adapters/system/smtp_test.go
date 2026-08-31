package system

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestSMTPMailerFailsClosedWhenSTARTTLSIsNotAdvertised(t *testing.T) {
	t.Parallel()
	certificate, caFile := smtpTestCertificate(t)
	server := newSMTPTestServer(t, smtpTestServerConfig{certificate: certificate})
	mailer := newSMTPTestMailer(t, server.address(), SMTPTLSStartTLS, "smtp.test", caFile)
	mailer.config.Username, mailer.config.Password = "smtp-user", "smtp-password"

	err := mailer.SendVerification(context.Background(), "owner@example.test", "submitter", "123456", time.Now())
	assertSMTPDeliveryFailure(t, err, "123456", "smtp-password")
	if transcript := server.transcript(); strings.Contains(transcript, "MAIL FROM") || strings.Contains(transcript, "123456") {
		t.Fatalf("verification content was sent before STARTTLS: %q", transcript)
	}
}

func TestSMTPMailerFailsClosedWhenSTARTTLSBecomesUnavailable(t *testing.T) {
	t.Parallel()
	certificate, caFile := smtpTestCertificate(t)
	server := newSMTPTestServer(t, smtpTestServerConfig{advertiseSTARTTLS: true, certificate: certificate, rejectSTARTTLS: true})
	mailer := newSMTPTestMailer(t, server.address(), SMTPTLSStartTLS, "smtp.test", caFile)
	mailer.config.Username, mailer.config.Password = "smtp-user", "smtp-password"

	err := mailer.SendVerification(context.Background(), "owner@example.test", "submitter", "123456", time.Now())
	assertSMTPDeliveryFailure(t, err, "123456", "smtp-password")
	if transcript := server.transcript(); strings.Contains(transcript, "MAIL FROM") || strings.Contains(transcript, "123456") {
		t.Fatalf("verification content was sent after STARTTLS rejection: %q", transcript)
	}
}

func TestSMTPMailerRejectsInvalidServerCertificate(t *testing.T) {
	t.Parallel()
	certificate, _ := smtpTestCertificate(t)
	server := newSMTPTestServer(t, smtpTestServerConfig{advertiseSTARTTLS: true, certificate: certificate})
	mailer := newSMTPTestMailer(t, server.address(), SMTPTLSStartTLS, "smtp.test", "")
	mailer.config.Username, mailer.config.Password = "smtp-user", "smtp-password"

	err := mailer.SendVerification(context.Background(), "owner@example.test", "submitter", "123456", time.Now())
	assertSMTPDeliveryFailure(t, err, "123456", "smtp-password")
	if transcript := server.transcript(); strings.Contains(transcript, "MAIL FROM") || strings.Contains(transcript, "123456") {
		t.Fatalf("verification content was sent after certificate validation failed: %q", transcript)
	}
}

func TestSMTPMailerDeliversOnlyAfterValidatedSTARTTLS(t *testing.T) {
	t.Parallel()
	certificate, caFile := smtpTestCertificate(t)
	server := newSMTPTestServer(t, smtpTestServerConfig{advertiseSTARTTLS: true, certificate: certificate})
	mailer := newSMTPTestMailer(t, server.address(), SMTPTLSStartTLS, "smtp.test", caFile)

	if err := mailer.SendVerification(context.Background(), "owner@example.test", "submitter", "123456", time.Now()); err != nil {
		t.Fatalf("SendVerification() error = %v", err)
	}
	transcript := server.transcript()
	if !strings.Contains(transcript, "STARTTLS") || !strings.Contains(transcript, "MAIL FROM") || !strings.Contains(transcript, "123456") {
		t.Fatalf("validated STARTTLS delivery transcript = %q", transcript)
	}
}

func TestSMTPMailerSupportsExplicitDevelopmentPlaintext(t *testing.T) {
	t.Parallel()
	server := newSMTPTestServer(t, smtpTestServerConfig{})
	mailer := newSMTPTestMailer(t, server.address(), SMTPTLSNone, "", "")

	if err := mailer.SendVerification(context.Background(), "owner@example.test", "submitter", "123456", time.Now()); err != nil {
		t.Fatalf("SendVerification() error = %v", err)
	}
	if transcript := server.transcript(); !strings.Contains(transcript, "MAIL FROM") || !strings.Contains(transcript, "123456") {
		t.Fatalf("development delivery transcript = %q", transcript)
	}
}

func TestSMTPMailerSupportsImplicitTLS(t *testing.T) {
	t.Parallel()
	certificate, caFile := smtpTestCertificate(t)
	server := newSMTPTestServer(t, smtpTestServerConfig{implicitTLS: true, certificate: certificate})
	mailer := newSMTPTestMailer(t, server.address(), SMTPTLSImplicit, "smtp.test", caFile)

	if err := mailer.SendVerification(context.Background(), "owner@example.test", "submitter", "123456", time.Now()); err != nil {
		t.Fatalf("SendVerification() error = %v", err)
	}
}

func newSMTPTestMailer(t *testing.T, address, mode, serverName, caFile string) *SMTPMailer {
	t.Helper()
	mailer, err := NewSMTPMailer(SMTPMailerConfig{
		Address:       address,
		From:          "no-reply@example.test",
		TLSMode:       mode,
		TLSServerName: serverName,
		TLSCAFile:     caFile,
	})
	if err != nil {
		t.Fatalf("NewSMTPMailer() error = %v", err)
	}
	return mailer
}

func assertSMTPDeliveryFailure(t *testing.T, err error, sensitive ...string) {
	t.Helper()
	if !errors.Is(err, errSMTPDelivery) {
		t.Fatalf("SendVerification() error = %v, want generic delivery failure", err)
	}
	for _, value := range sensitive {
		if strings.Contains(err.Error(), value) {
			t.Fatalf("SendVerification() exposed sensitive value %q", value)
		}
	}
}

type smtpTestServerConfig struct {
	advertiseSTARTTLS bool
	rejectSTARTTLS    bool
	implicitTLS       bool
	certificate       tls.Certificate
}

type smtpTestServer struct {
	listener net.Listener
	config   smtpTestServerConfig
	done     chan struct{}
	mu       sync.Mutex
	lines    []string
}

func newSMTPTestServer(t *testing.T, config smtpTestServerConfig) *smtpTestServer {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &smtpTestServer{listener: listener, config: config, done: make(chan struct{})}
	go server.serve()
	t.Cleanup(func() {
		_ = listener.Close()
		select {
		case <-server.done:
		case <-time.After(time.Second):
			t.Error("SMTP test server did not stop")
		}
	})
	return server
}

func (server *smtpTestServer) address() string { return server.listener.Addr().String() }

func (server *smtpTestServer) transcript() string {
	server.mu.Lock()
	defer server.mu.Unlock()
	return strings.Join(server.lines, "\n")
}

func (server *smtpTestServer) record(line string) {
	server.mu.Lock()
	defer server.mu.Unlock()
	server.lines = append(server.lines, line)
}

func (server *smtpTestServer) serve() {
	defer close(server.done)
	connection, err := server.listener.Accept()
	if err != nil {
		return
	}
	defer connection.Close()
	if server.config.implicitTLS {
		connection = tls.Server(connection, &tls.Config{Certificates: []tls.Certificate{server.config.certificate}, MinVersion: tls.VersionTLS12})
		if err := connection.(*tls.Conn).Handshake(); err != nil {
			return
		}
	}
	server.session(connection)
}

func (server *smtpTestServer) session(connection net.Conn) {
	reader := bufio.NewReader(connection)
	writer := bufio.NewWriter(connection)
	writeSMTPResponse(writer, "220 smtp.test ESMTP ready")
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		server.record(line)
		command := strings.ToUpper(strings.Fields(line)[0])
		switch command {
		case "EHLO", "HELO":
			if server.config.advertiseSTARTTLS {
				writeSMTPResponse(writer, "250-smtp.test", "250 STARTTLS")
			} else {
				writeSMTPResponse(writer, "250 smtp.test")
			}
		case "STARTTLS":
			if server.config.rejectSTARTTLS {
				writeSMTPResponse(writer, "454 TLS temporarily unavailable")
				continue
			}
			writeSMTPResponse(writer, "220 ready to start TLS")
			tlsConnection := tls.Server(connection, &tls.Config{Certificates: []tls.Certificate{server.config.certificate}, MinVersion: tls.VersionTLS12})
			if err := tlsConnection.Handshake(); err != nil {
				return
			}
			connection = tlsConnection
			reader = bufio.NewReader(connection)
			writer = bufio.NewWriter(connection)
		case "MAIL", "RCPT":
			writeSMTPResponse(writer, "250 accepted")
		case "DATA":
			writeSMTPResponse(writer, "354 end with dot")
			for {
				dataLine, err := reader.ReadString('\n')
				if err != nil {
					return
				}
				if dataLine == ".\r\n" {
					break
				}
				server.record(strings.TrimRight(dataLine, "\r\n"))
			}
			writeSMTPResponse(writer, "250 queued")
		case "QUIT":
			writeSMTPResponse(writer, "221 bye")
			return
		default:
			writeSMTPResponse(writer, "502 unsupported")
		}
	}
}

func writeSMTPResponse(writer *bufio.Writer, lines ...string) {
	for _, line := range lines {
		_, _ = writer.WriteString(line + "\r\n")
	}
	_ = writer.Flush()
}

func smtpTestCertificate(t *testing.T) (tls.Certificate, string) {
	t.Helper()
	const serverName = "smtp.test"
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: serverName},
		DNSNames:              []string{serverName},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	keyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	certificatePEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes})
	certificate, err := tls.X509KeyPair(certificatePEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	caFile := filepath.Join(t.TempDir(), "smtp-ca.pem")
	if err := os.WriteFile(caFile, certificatePEM, 0o600); err != nil {
		t.Fatal(err)
	}
	return certificate, caFile
}
