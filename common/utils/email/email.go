package email

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
)

type EmailConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	FromName string
	Subject  string
}

func SendQQEmail(cfg EmailConfig, toEmail, code string) error {
	// 组装邮件内容
	// header
	header := make(map[string]string)
	header["From"] = fmt.Sprintf("%s <%s>", cfg.FromName, cfg.Username)
	header["To"] = toEmail
	header["Subject"] = cfg.Subject
	header["Content-Type"] = "text/plain; charset=UTF-8"

	message := ""
	for k, v := range header {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + fmt.Sprintf("您的验证码是：%s，请在5分钟内使用。", code)

	// 连接到SMTP服务器
	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)

	// QQ邮箱使用SSL端口465，需要使用tls连接
	if cfg.Port == 465 {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         cfg.Host,
		}

		conn, err := tls.Dial("tcp", fmt.Sprintf("%s:%d", cfg.Host, cfg.Port), tlsConfig)
		if err != nil {
			return err
		}

		client, err := smtp.NewClient(conn, cfg.Host)
		if err != nil {
			return err
		}
		defer client.Quit()

		if err = client.Auth(auth); err != nil {
			return err
		}

		if err = client.Mail(cfg.Username); err != nil {
			return err
		}

		if err = client.Rcpt(toEmail); err != nil {
			return err
		}

		w, err := client.Data()
		if err != nil {
			return err
		}

		_, err = w.Write([]byte(message))
		if err != nil {
			return err
		}

		err = w.Close()
		if err != nil {
			return err
		}

		return nil
	}

	// 非SSL端口（通常不会走到这里，因为配置的是465）
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	return smtp.SendMail(addr, auth, cfg.Username, []string{toEmail}, []byte(message))
}
