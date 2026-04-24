package services

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"net/smtp"
	"os"
	"strings"

	"github.com/zatrano/framework/configs/logconfig"

	"go.uber.org/zap"
)

type IMailService interface {
	SendMail(to, subject, body string) error
	SendTemplateMail(to, subject, tmplName string, data map[string]interface{}) error
}

type MailService struct {
	host        string
	port        string
	username    string
	password    string
	fromAddress string
	fromName    string
	encryption  string
}

// emailTemplates — yerleşik HTML e-posta şablonları.
// Ayrı dosyaya gerek kalmadan tek yerden yönetilir.
var emailTemplates = map[string]string{
	"verification": `<!DOCTYPE html>
<html lang="tr">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1"></head>
<body style="font-family:Arial,sans-serif;background:#f4f4f4;margin:0;padding:0">
  <table width="100%" cellpadding="0" cellspacing="0" style="background:#f4f4f4;padding:40px 0">
    <tr><td align="center">
      <table width="600" cellpadding="0" cellspacing="0" style="background:#fff;border-radius:8px;overflow:hidden;box-shadow:0 2px 8px rgba(0,0,0,.1)">
        <tr><td style="background:#4f46e5;padding:32px;text-align:center">
          <h1 style="color:#fff;margin:0;font-size:24px">{{.SiteName}}</h1>
        </td></tr>
        <tr><td style="padding:40px 32px">
          <h2 style="color:#1f2937;margin:0 0 16px">E-posta Adresinizi Doğrulayın</h2>
          <p style="color:#6b7280;line-height:1.6;margin:0 0 24px">Merhaba,</p>
          <p style="color:#6b7280;line-height:1.6;margin:0 0 32px">
            Hesabınızı aktifleştirmek için aşağıdaki butona tıklayın.
            Bu link <strong>{{.ExpiryHours}} saat</strong> süreyle geçerlidir.
          </p>
          <div style="text-align:center;margin:0 0 32px">
            <a href="{{.Link}}" style="background:#4f46e5;color:#fff;padding:14px 32px;border-radius:6px;text-decoration:none;font-weight:600;display:inline-block">
              E-postamı Doğrula
            </a>
          </div>
          <p style="color:#9ca3af;font-size:13px;margin:0">
            Butona tıklayamıyorsanız şu adresi tarayıcınıza yapıştırın:<br>
            <a href="{{.Link}}" style="color:#4f46e5;word-break:break-all">{{.Link}}</a>
          </p>
        </td></tr>
        <tr><td style="background:#f9fafb;padding:24px 32px;text-align:center">
          <p style="color:#9ca3af;font-size:12px;margin:0">
            Bu e-postayı siz talep etmediyseniz güvenle yok sayabilirsiniz.
          </p>
        </td></tr>
      </table>
    </td></tr>
  </table>
</body>
</html>`,

	"password_reset": `<!DOCTYPE html>
<html lang="tr">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1"></head>
<body style="font-family:Arial,sans-serif;background:#f4f4f4;margin:0;padding:0">
  <table width="100%" cellpadding="0" cellspacing="0" style="background:#f4f4f4;padding:40px 0">
    <tr><td align="center">
      <table width="600" cellpadding="0" cellspacing="0" style="background:#fff;border-radius:8px;overflow:hidden;box-shadow:0 2px 8px rgba(0,0,0,.1)">
        <tr><td style="background:#dc2626;padding:32px;text-align:center">
          <h1 style="color:#fff;margin:0;font-size:24px">{{.SiteName}}</h1>
        </td></tr>
        <tr><td style="padding:40px 32px">
          <h2 style="color:#1f2937;margin:0 0 16px">Şifre Sıfırlama</h2>
          <p style="color:#6b7280;line-height:1.6;margin:0 0 24px">Merhaba,</p>
          <p style="color:#6b7280;line-height:1.6;margin:0 0 32px">
            Şifrenizi sıfırlamak için aşağıdaki butona tıklayın.
            Bu link <strong>{{.ExpiryHours}} saat</strong> sonra geçersiz olacaktır.
          </p>
          <div style="text-align:center;margin:0 0 32px">
            <a href="{{.Link}}" style="background:#dc2626;color:#fff;padding:14px 32px;border-radius:6px;text-decoration:none;font-weight:600;display:inline-block">
              Şifremi Sıfırla
            </a>
          </div>
          <p style="color:#9ca3af;font-size:13px;margin:0">
            Butona tıklayamıyorsanız şu adresi tarayıcınıza yapıştırın:<br>
            <a href="{{.Link}}" style="color:#dc2626;word-break:break-all">{{.Link}}</a>
          </p>
        </td></tr>
        <tr><td style="background:#f9fafb;padding:24px 32px;text-align:center">
          <p style="color:#9ca3af;font-size:12px;margin:0">
            Bu isteği siz yapmadıysanız bu e-postayı güvenle yok sayabilirsiniz.
            Şifreniz değiştirilmeyecektir.
          </p>
        </td></tr>
      </table>
    </td></tr>
  </table>
</body>
</html>`,
}

func NewMailService() IMailService {
	encryption := strings.ToLower(getEnvWithDefault("MAIL_ENCRYPTION", "tls"))
	port := getEnvWithDefault("MAIL_PORT", "")

	if port == "" {
		switch encryption {
		case "ssl":
			port = "465"
		case "tls":
			port = "587"
		default:
			port = "25"
		}
	}

	return &MailService{
		host:        getEnvWithDefault("MAIL_HOST", "localhost"),
		port:        port,
		username:    getEnvWithDefault("MAIL_USERNAME", ""),
		password:    getEnvWithDefault("MAIL_PASSWORD", ""),
		fromAddress: getEnvWithDefault("MAIL_FROM_ADDRESS", ""),
		fromName:    getEnvWithDefault("MAIL_FROM_NAME", ""),
		encryption:  encryption,
	}
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (m *MailService) SendMail(to, subject, body string) error {
	if to == "" {
		return fmt.Errorf("alıcı e-posta adresi (to) boş olamaz")
	}
	if m.fromAddress == "" {
		return fmt.Errorf("gönderen e-posta adresi (MAIL_FROM_ADDRESS) tanımlanmamış")
	}

	message, err := m.buildMessage(to, subject, body)
	if err != nil {
		return fmt.Errorf("e-posta mesajı oluşturulamadı: %w", err)
	}

	err = m.send(to, message)
	if err != nil {
		logconfig.Log.Error("E-posta gönderimi başarısız oldu", zap.Error(err), zap.String("to", to))
		return err
	}

	logconfig.Log.Info("E-posta başarıyla gönderildi", zap.String("to", to))
	return nil
}

// SendTemplateMail — adlandırılmış HTML şablonu ile e-posta gönderir.
// tmplName: "verification" veya "password_reset"
func (m *MailService) SendTemplateMail(to, subject, tmplName string, data map[string]interface{}) error {
	if to == "" {
		return fmt.Errorf("alıcı e-posta adresi (to) boş olamaz")
	}
	if m.fromAddress == "" {
		return fmt.Errorf("gönderen e-posta adresi (MAIL_FROM_ADDRESS) tanımlanmamış")
	}

	tmplSrc, ok := emailTemplates[tmplName]
	if !ok {
		return fmt.Errorf("bilinmeyen e-posta şablonu: %s", tmplName)
	}

	tmpl, err := template.New(tmplName).Parse(tmplSrc)
	if err != nil {
		return fmt.Errorf("e-posta şablonu ayrıştırılamadı: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("e-posta şablonu çalıştırılamadı: %w", err)
	}

	message, err := m.buildMessage(to, subject, buf.String())
	if err != nil {
		return fmt.Errorf("e-posta mesajı oluşturulamadı: %w", err)
	}

	if err := m.send(to, message); err != nil {
		logconfig.Log.Error("Şablonlu e-posta gönderimi başarısız", zap.Error(err), zap.String("to", to), zap.String("template", tmplName))
		return err
	}

	logconfig.Log.Info("Şablonlu e-posta gönderildi", zap.String("to", to), zap.String("template", tmplName))
	return nil
}

func (m *MailService) buildMessage(to, subject, body string) ([]byte, error) {
	if subject == "" {
		subject = "(Konu Belirtilmemiş)"
	}

	fromHeader := m.fromAddress
	if m.fromName != "" {
		fromHeader = fmt.Sprintf("\"%s\" <%s>", m.fromName, m.fromAddress)
	}

	header := fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"Content-Type: text/html; charset=UTF-8\r\n\r\n",
		fromHeader, to, subject)

	return []byte(header + body), nil
}

func (m *MailService) send(to string, message []byte) error {
	address := fmt.Sprintf("%s:%s", m.host, m.port)
	auth := smtp.PlainAuth("", m.username, m.password, m.host)

	switch m.encryption {
	case "tls":
		client, err := smtp.Dial(address)
		if err != nil {
			return fmt.Errorf("SMTP sunucusuna bağlanılamadı: %w", err)
		}
		defer client.Quit()

		if err := client.StartTLS(&tls.Config{ServerName: m.host, InsecureSkipVerify: false}); err != nil {
			return fmt.Errorf("STARTTLS başlatılamadı: %w", err)
		}

		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("kimlik doğrulama başarısız: %w", err)
		}

		return sendMailWithClient(client, m.fromAddress, to, message)

	case "ssl":
		tlsConfig := &tls.Config{ServerName: m.host, InsecureSkipVerify: false}
		conn, err := tls.Dial("tcp", address, tlsConfig)
		if err != nil {
			return fmt.Errorf("SSL ile TLS bağlantısı kurulamadı: %w", err)
		}

		client, err := smtp.NewClient(conn, m.host)
		if err != nil {
			return fmt.Errorf("SSL bağlantısı üzerinden SMTP istemcisi oluşturulamadı: %w", err)
		}
		defer client.Quit()

		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("kimlik doğrulama başarısız: %w", err)
		}

		return sendMailWithClient(client, m.fromAddress, to, message)

	default:
		return smtp.SendMail(address, auth, m.fromAddress, []string{to}, message)
	}
}

func sendMailWithClient(client *smtp.Client, from, to string, message []byte) error {
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("SMTP göndericisi (%s) ayarlanamadı: %w", from, err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("SMTP alıcısı (%s) ayarlanamadı: %w", to, err)
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA komutu başlatılamadı: %w", err)
	}
	if _, err := writer.Write(message); err != nil {
		writer.Close()
		return fmt.Errorf("mesaj verisi yazılamadı: %w", err)
	}
	return writer.Close()
}

var _ IMailService = (*MailService)(nil)
