package email

import (
	"fmt"
	"net/smtp"

	"smart-live-chats/internal/config"
)

type Service struct {
	host     string
	port     string
	user     string
	password string
	from     string
}

func NewService(cfg *config.Config) *Service {
	return &Service{
		host:     cfg.SMTPHost,
		port:     cfg.SMTPPort,
		user:     cfg.SMTPUser,
		password: cfg.SMTPPassword,
		from:     cfg.SMTPFrom,
	}
}

func (s *Service) SendVerificationCode(toEmail, code string) error {
	subject := "AIChatsHub - Email Verification"
	body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family: 'Segoe UI', Arial, sans-serif; margin: 0; padding: 0; background: #f5f7fa;">
  <div style="max-width: 480px; margin: 40px auto; background: #fff; border-radius: 12px; overflow: hidden; box-shadow: 0 2px 12px rgba(0,0,0,0.08);">
    <div style="background: linear-gradient(135deg, #1a3a8f, #3b82f6); padding: 32px; text-align: center;">
      <h1 style="color: #fff; margin: 0; font-size: 22px;">AIChatsHub</h1>
    </div>
    <div style="padding: 32px; text-align: center;">
      <p style="color: #374151; font-size: 16px; margin-bottom: 8px;">Your verification code is:</p>
      <div style="font-size: 36px; font-weight: 700; letter-spacing: 8px; color: #1a3a8f; background: #f0f4ff; border-radius: 8px; padding: 16px; margin: 16px 0;">%s</div>
      <p style="color: #6b7280; font-size: 14px;">This code expires in 10 minutes.</p>
    </div>
  </div>
</body>
</html>`, code)

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		s.from, toEmail, subject, body)

	auth := smtp.PlainAuth("", s.user, s.password, s.host)
	addr := fmt.Sprintf("%s:%s", s.host, s.port)

	if err := smtp.SendMail(addr, auth, s.from, []string{toEmail}, []byte(msg)); err != nil {
		return fmt.Errorf("smtp send to %s from %s via %s: %w", toEmail, s.from, addr, err)
	}
	return nil
}
