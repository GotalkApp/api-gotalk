package mailer

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/smtp"
)

// Config holds SMTP configuration
type Config struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
	FromName string
}

// Mailer handles sending emails
type Mailer struct {
	config Config
}

// New creates a new Mailer instance
func New(cfg Config) *Mailer {
	return &Mailer{config: cfg}
}

// SendOTP sends an OTP verification email
func (m *Mailer) SendOTP(toEmail, username, code string, expiryMinutes int) error {
	subject := "GoTalk - Verify your email address"

	body, err := m.renderOTPTemplate(username, code, expiryMinutes)
	if err != nil {
		return fmt.Errorf("failed to render email template: %w", err)
	}

	return m.send(toEmail, subject, body)
}

// SendPasswordReset sends a password reset OTP email
func (m *Mailer) SendPasswordReset(toEmail, username, code string, expiryMinutes int) error {
	subject := "GoTalk - Reset your password"

	body, err := m.renderPasswordResetTemplate(username, code, expiryMinutes)
	if err != nil {
		return fmt.Errorf("failed to render email template: %w", err)
	}

	return m.send(toEmail, subject, body)
}

// send delivers an email via SMTP
func (m *Mailer) send(to, subject, htmlBody string) error {
	addr := fmt.Sprintf("%s:%s", m.config.Host, m.config.Port)

	headers := map[string]string{
		"From":         fmt.Sprintf("%s <%s>", m.config.FromName, m.config.From),
		"To":           to,
		"Subject":      subject,
		"MIME-Version": "1.0",
		"Content-Type": "text/html; charset=\"utf-8\"",
	}

	var msg bytes.Buffer
	for k, v := range headers {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msg.WriteString("\r\n")
	msg.WriteString(htmlBody)

	var auth smtp.Auth
	if m.config.Username != "" && m.config.Password != "" {
		auth = smtp.PlainAuth("", m.config.Username, m.config.Password, m.config.Host)
	}

	err := smtp.SendMail(addr, auth, m.config.From, []string{to}, msg.Bytes())
	if err != nil {
		log.Printf("❌ Failed to send email to %s: %v", to, err)
		return fmt.Errorf("failed to send email: %w", err)
	}

	log.Printf("📧 Email sent to %s: %s", to, subject)
	return nil
}

// renderOTPTemplate returns the HTML body for OTP verification email
func (m *Mailer) renderOTPTemplate(username, code string, expiryMinutes int) (string, error) {
	tmpl := `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="margin:0;padding:0;background-color:#0f0f23;font-family:'Segoe UI',Tahoma,Geneva,Verdana,sans-serif;">
    <div style="max-width:500px;margin:40px auto;background:linear-gradient(135deg,#1a1a2e 0%,#16213e 100%);border-radius:16px;overflow:hidden;border:1px solid rgba(99,102,241,0.2);">
        <!-- Header -->
        <div style="background:linear-gradient(135deg,#6366f1 0%,#8b5cf6 100%);padding:32px;text-align:center;">
            <h1 style="color:#fff;margin:0;font-size:28px;font-weight:700;">🚀 GoTalk</h1>
            <p style="color:rgba(255,255,255,0.85);margin:8px 0 0;font-size:14px;">Email Verification</p>
        </div>

        <!-- Body -->
        <div style="padding:32px;">
            <p style="color:#e2e8f0;font-size:16px;line-height:1.6;margin:0 0 24px;">
                Hi <strong style="color:#a78bfa;">{{.Username}}</strong>,
            </p>
            <p style="color:#94a3b8;font-size:14px;line-height:1.6;margin:0 0 24px;">
                Your verification code is:
            </p>

            <!-- OTP Code -->
            <div style="background:rgba(99,102,241,0.1);border:2px dashed rgba(99,102,241,0.4);border-radius:12px;padding:24px;text-align:center;margin:0 0 24px;">
                <span style="font-size:36px;font-weight:800;letter-spacing:8px;color:#818cf8;font-family:'Courier New',monospace;">{{.Code}}</span>
            </div>

            <p style="color:#64748b;font-size:13px;line-height:1.5;margin:0 0 8px;">
                ⏰ This code expires in <strong style="color:#f59e0b;">{{.ExpiryMinutes}} minutes</strong>.
            </p>
            <p style="color:#64748b;font-size:13px;line-height:1.5;margin:0;">
                If you didn't create a GoTalk account, please ignore this email.
            </p>
        </div>

        <!-- Footer -->
        <div style="padding:16px 32px;border-top:1px solid rgba(99,102,241,0.1);text-align:center;">
            <p style="color:#475569;font-size:12px;margin:0;">© 2026 GoTalk. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`

	t, err := template.New("otp").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = t.Execute(&buf, map[string]interface{}{
		"Username":      username,
		"Code":          code,
		"ExpiryMinutes": expiryMinutes,
	})
	return buf.String(), err
}

// renderPasswordResetTemplate returns the HTML body for password reset email
func (m *Mailer) renderPasswordResetTemplate(username, code string, expiryMinutes int) (string, error) {
	tmpl := `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="margin:0;padding:0;background-color:#0f0f23;font-family:'Segoe UI',Tahoma,Geneva,Verdana,sans-serif;">
    <div style="max-width:500px;margin:40px auto;background:linear-gradient(135deg,#1a1a2e 0%,#16213e 100%);border-radius:16px;overflow:hidden;border:1px solid rgba(239,68,68,0.2);">
        <!-- Header -->
        <div style="background:linear-gradient(135deg,#ef4444 0%,#dc2626 100%);padding:32px;text-align:center;">
            <h1 style="color:#fff;margin:0;font-size:28px;font-weight:700;">🔐 GoTalk</h1>
            <p style="color:rgba(255,255,255,0.85);margin:8px 0 0;font-size:14px;">Password Reset</p>
        </div>

        <!-- Body -->
        <div style="padding:32px;">
            <p style="color:#e2e8f0;font-size:16px;line-height:1.6;margin:0 0 24px;">
                Hi <strong style="color:#fca5a5;">{{.Username}}</strong>,
            </p>
            <p style="color:#94a3b8;font-size:14px;line-height:1.6;margin:0 0 24px;">
                We received a request to reset your password. Use this code:
            </p>

            <!-- OTP Code -->
            <div style="background:rgba(239,68,68,0.1);border:2px dashed rgba(239,68,68,0.4);border-radius:12px;padding:24px;text-align:center;margin:0 0 24px;">
                <span style="font-size:36px;font-weight:800;letter-spacing:8px;color:#f87171;font-family:'Courier New',monospace;">{{.Code}}</span>
            </div>

            <p style="color:#64748b;font-size:13px;line-height:1.5;margin:0 0 8px;">
                ⏰ This code expires in <strong style="color:#f59e0b;">{{.ExpiryMinutes}} minutes</strong>.
            </p>
            <p style="color:#64748b;font-size:13px;line-height:1.5;margin:0;">
                If you didn't request a password reset, please ignore this email and your password will remain unchanged.
            </p>
        </div>

        <!-- Footer -->
        <div style="padding:16px 32px;border-top:1px solid rgba(239,68,68,0.1);text-align:center;">
            <p style="color:#475569;font-size:12px;margin:0;">© 2026 GoTalk. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`

	t, err := template.New("reset").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = t.Execute(&buf, map[string]interface{}{
		"Username":      username,
		"Code":          code,
		"ExpiryMinutes": expiryMinutes,
	})
	return buf.String(), err
}
