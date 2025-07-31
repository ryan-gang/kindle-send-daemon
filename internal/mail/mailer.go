package mail

import (
	"fmt"
	"os"
	"time"

	"github.com/ryan-gang/kindle-send-daemon/internal/util"

	gomail "gopkg.in/mail.v2"
)

func (s *SMTPMailSender) Send(files []string, timeout int) error {
	cfg := s.cfg
	msg := gomail.NewMessage()
	msg.SetHeader("From", cfg.GetSender())
	msg.SetHeader("To", cfg.GetReceiver())

	msg.SetBody("text/plain", "")

	attachedFiles := make([]string, 0)
	for _, file := range files {
		_, err := os.Stat(file)
		if err != nil {
			util.LogErrorf(util.FileError, "accessing file", "couldn't find file %s", file)
			continue
		} else {
			msg.Attach(file)
			attachedFiles = append(attachedFiles, file)
		}
	}
	if len(attachedFiles) == 0 {
		util.Cyan.Println("No files to send")
		return fmt.Errorf("no valid files to send")
	}

	dialer := gomail.NewDialer(cfg.GetServer(), cfg.GetPort(), cfg.GetSender(), cfg.GetPassword())
	dialer.Timeout = time.Duration(timeout) * time.Second
	util.CyanBold.Println("Sending mail")
	util.Cyan.Println("Mail timeout : ", dialer.Timeout.String())
	util.Cyan.Println("Following files will be sent :")
	for i, file := range attachedFiles {
		util.Cyan.Printf("%d. %s\n", i+1, file)
	}

	if err := dialer.DialAndSend(msg); err != nil {
		util.LogError(util.MailError, "sending mail", err)
		return fmt.Errorf("failed to send mail: %w", err)
	}

	util.GreenBold.Printf("Mailed %d files to %s", len(attachedFiles), cfg.GetReceiver())
	return nil
}
