package auth

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/smtp"
	"os"
	"strconv"
	"time"
)

// EmailService handles sending verification emails.
// In dev mode (SMTP_HOST not set), codes are printed to stdout.
type EmailService struct {
	host     string
	port     int
	user     string
	password string
	from     string
	devMode  bool
}

// IsDevMode returns true when SMTP is not configured (codes are logged instead of sent).
func (s *EmailService) IsDevMode() bool { return s.devMode }

// NewEmailService creates an EmailService from SMTP_* environment variables.
// If SMTP_HOST is empty or SMTP_PORT is 0, dev mode is enabled.
func NewEmailService() *EmailService {
	host := os.Getenv("SMTP_HOST")
	portStr := os.Getenv("SMTP_PORT")
	port, _ := strconv.Atoi(portStr)
	user := os.Getenv("SMTP_USER")
	password := os.Getenv("SMTP_PASSWORD")
	from := os.Getenv("SMTP_FROM")
	if from == "" {
		from = "noreply@resumegenius.com"
	}
	// Dev mode only when SMTP config is incomplete — any missing piece disables SMTP.
	devMode := host == "" || port == 0 || user == "" || password == ""
	return &EmailService{
		host: host, port: port, user: user,
		password: password, from: from,
		devMode: devMode,
	}
}

// GenerateCode returns a cryptographically random 6-digit code as a string.
func GenerateCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", fmt.Errorf("generate code: %w", err)
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// SendVerificationCode sends a 6-digit verification code to the given email.
// In dev mode, the code is printed to stdout.
// In production mode, a 10-second timeout is enforced on SMTP connection.
// If SMTP fails in production mode, the error is returned to the caller.
func (s *EmailService) SendVerificationCode(to, code string) error {
	if s.devMode {
		log.Printf("[DEV MODE] Verification code for %s: %s", to, code)
		return nil
	}

	msg := []byte(fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: =?utf-8?B?%s?=\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: text/plain; charset=UTF-8\r\n\r\n"+
			"您的验证码是：%s\r\n\r\n"+
			"验证码 15 分钟内有效，请勿告知他人。\r\n"+
			"如非本人操作，请忽略此邮件。",
		s.from, to, base64Encode("ResumeGenius 邮箱验证"),
		code,
	))

	addr := fmt.Sprintf("%s:%d", s.host, s.port)

	var client *smtp.Client
	if s.port == 465 {
		// Implicit TLS (SMTPS) — TLS from the start
		tlsConn, err := tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", addr, &tls.Config{ServerName: s.host})
		if err != nil {
			log.Printf("[SMTP] TLS connection to %s failed for %s: %v", addr, to, err)
			return fmt.Errorf("smtp tls connect: %w", err)
		}
		defer tlsConn.Close()
		client, err = smtp.NewClient(tlsConn, s.host)
		if err != nil {
			log.Printf("[SMTP] failed to create client for %s: %v", to, err)
			return fmt.Errorf("smtp create client: %w", err)
		}
		defer client.Quit()
	} else {
		conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
		if err != nil {
			log.Printf("[SMTP] failed to connect to %s for %s: %v", addr, to, err)
			return fmt.Errorf("smtp connect: %w", err)
		}
		defer conn.Close()
		client, err = smtp.NewClient(conn, s.host)
		if err != nil {
			log.Printf("[SMTP] failed to create client for %s: %v", to, err)
			return fmt.Errorf("smtp create client: %w", err)
		}
		defer client.Quit()
		// Try STARTTLS for non-465 ports if server supports it
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: s.host}); err != nil {
				log.Printf("[SMTP] StartTLS failed for %s: %v", to, err)
				return fmt.Errorf("smtp starttls: %w", err)
			}
		}
	}

	auth := smtp.PlainAuth("", s.user, s.password, s.host)
	if err := client.Auth(auth); err != nil {
		log.Printf("[SMTP] auth failed for %s: %v", to, err)
		return fmt.Errorf("smtp auth: %w", err)
	}
	if err := client.Mail(s.from); err != nil {
		log.Printf("[SMTP] MAIL FROM failed for %s: %v", to, err)
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		log.Printf("[SMTP] RCPT TO failed for %s: %v", to, err)
		return fmt.Errorf("smtp rcpt to: %w", err)
	}
	wc, err := client.Data()
	if err != nil {
		log.Printf("[SMTP] DATA failed for %s: %v", to, err)
		return fmt.Errorf("smtp data: %w", err)
	}
	_, writeErr := wc.Write(msg)
	closeErr := wc.Close()
	if writeErr != nil {
		log.Printf("[SMTP] write failed for %s: %v", to, writeErr)
		return fmt.Errorf("smtp write: %w", writeErr)
	}
	if closeErr != nil {
		log.Printf("[SMTP] close failed for %s: %v", to, closeErr)
		return fmt.Errorf("smtp close data: %w", closeErr)
	}

	log.Printf("[SMTP] Verification code sent to %s", to)
	return nil
}

// base64Encode returns the base64-encoded string for use in RFC 2047 encoded-words.
func base64Encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}
