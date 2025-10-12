package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"samuelemusiani/sasso/server/config"
	"samuelemusiani/sasso/server/db"
	"time"

	"github.com/wneessen/go-mail"
)

var (
	logger *slog.Logger = nil
	email  string

	emailClient *mail.Client = nil

	workerContext    context.Context
	workerCancelFunc context.CancelFunc
	workerReturnChan chan error = make(chan error, 1)
)

const telegramAPIURL = "https://api.telegram.org/bot"

type telegramMessage struct {
	ChatID string `json:"chat_id"`
	Text   string `json:"text"`
}

type notification struct {
	UserID  uint
	Subject string
	Body    string
}

func Init(l *slog.Logger, c config.Email) error {
	logger = l

	if !c.Enabled {
		slog.Info("Email notifications are disabled")
		return nil
	}

	email = c.Username

	var err error
	emailClient, err = mail.NewClient(c.SMTPServer, mail.WithSMTPAuth(mail.SMTPAuthPlain), mail.WithUsername(c.Username), mail.WithPassword(c.Password), mail.WithSSL(), mail.WithPort(465))
	if err != nil {
		slog.Error("Failed to create mail client", "error", err)
		return err
	}

	return nil
}

func StartWorker() {
	workerContext, workerCancelFunc = context.WithCancel(context.Background())
	go func() {
		workerReturnChan <- worker(workerContext)
		close(workerReturnChan)
	}()
}

func ShutdownWorker() error {
	if workerCancelFunc != nil {
		workerCancelFunc()
	}
	var err error
	if workerReturnChan != nil {
		err = <-workerReturnChan
	}
	if err != nil && err != context.Canceled {
		return err
	} else {
		return nil
	}
}

func worker(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(10 * time.Second):
		// Just a small delay to let other components start
	}

	logger.Info("Notification worker started")

	timeToWait := time.Second * 30

	for {
		select {
		case <-ctx.Done():
			logger.Info("Notification worker shutting down")
			return ctx.Err()
		case <-time.After(timeToWait):
			logger.Debug("Checking for new notifications to send")
		}

		now := time.Now()

		sendNotifications()

		elapsed := time.Since(now)
		if elapsed < 30*time.Second {
			timeToWait = 30*time.Second - elapsed
		} else {
			timeToWait = 0
		}
	}
}

func sendNotifications() {
	ntfs, err := db.GetPendingNotifications()
	if err != nil {
		logger.Error("Failed to get pending notifications", "error", err)
		return
	}

	for _, n := range ntfs {
		tmpN := notification{
			UserID:  n.UserID,
			Subject: n.Subject,
			Body:    n.Body,
		}

		err := sendEmail(&tmpN)
		if err != nil {
			logger.Error("Failed to send notification email", "userID", n.UserID, "error", err)
		}

		err = sendTelegram(&tmpN)
		if err != nil {
			logger.Error("Failed to send notification telegram", "userID", n.UserID, "error", err)
		}

		err = db.SetNotificationAsSent(n.ID)
		if err != nil {
			logger.Error("Failed to set notification as sent", "notificationID", n.ID, "error", err)
		}
	}
}

func (n *notification) save() error {
	return db.InsertNotification(n.UserID, n.Subject, n.Body)
}

func SendPortForwardNotification(userID uint, pf db.PortForward) error {
	t := `Your port forwarding request has been %s!
Outside port: %d
Destination port: %d
Destination IP: %s
`

	body := fmt.Sprintf(t, map[bool]string{true: "approved", false: "disabled"}[pf.Approved], pf.OutPort, pf.DestPort, pf.DestIP)

	n := &notification{
		UserID:  userID,
		Subject: "Port Forwarding Status Update",
		Body:    body,
	}
	err := n.save()
	if err != nil {
		logger.Error("Failed to save port forward notification", "userID", userID, "error", err)
		return err
	}
	return nil
}

func sendEmail(n *notification) error {
	if n.UserID == 0 {
		return sendBulkEmail(n)
	} else {
		return sendSingleEmail(n)
	}
}

func sendTelegram(n *notification) error {
	if n.UserID == 0 {
		return sendBulkTelegram(n)
	} else {
		return sendSingleTelegram(n)
	}
}

func sendSingleEmail(n *notification) error {
	user, err := db.GetUserByID(n.UserID)
	if err != nil {
		logger.Error("Failed to get user for notification", "userID", n.UserID, "error", err)
		return err
	}

	message := mail.NewMsg()
	if err := message.From(email); err != nil {
		logger.Error("Invalid 'From' address", "error", err)
		return err
	}
	if err := message.To(user.Email); err != nil {
		logger.Error("Invalid 'To' address", "error", err)
		return err
	}
	message.Subject(n.Subject)
	message.SetBodyString(mail.TypeTextPlain, n.Body)

	if err := emailClient.DialAndSend(message); err != nil {
		logger.Error("Failed to send email", "error", err)
		return err
	}

	logger.Debug("Email sent successfully", "to", user.Email)
	return nil
}

func sendBulkEmail(n *notification) error {
	emails, err := db.GetAllUserEmails()
	if err != nil {
		logger.Error("Failed to get user for notification", "userID", n.UserID, "error", err)
		return err
	}

	messages := make([]*mail.Msg, 0, len(emails))
	for _, e := range emails {
		message := mail.NewMsg()
		if err := message.From(email); err != nil {
			logger.Error("Invalid 'From' address", "error", err)
			return err
		}
		if err := message.To(e); err != nil {
			logger.Error("Invalid 'To' address", "error", err)
			return err
		}
		message.Subject(n.Subject)
		message.SetBodyString(mail.TypeTextPlain, n.Body)

		message.SetMessageID()
		message.SetBulk()
		message.SetDate()

		messages = append(messages, message)
	}

	if err := emailClient.DialAndSend(messages...); err != nil {
		logger.Error("Failed to send emails", "error", err)
		return err
	}

	logger.Debug("Emails sent successfully everyone")
	return nil
}

func sendSingleTelegram(n *notification) error {
	bots, err := db.GetTelegramBotsByUserID(n.UserID)
	if err != nil {
		logger.Error("Failed to get telegram bots for user", "userID", n.UserID, "error", err)
		return err
	}

	for _, bot := range bots {
		err := sendTelegramMessage(&bot, n.Body)
		if err != nil {
			logger.Error("Failed to send telegram message", "userID", n.UserID, "botID", bot.ID, "error", err)
		}
		time.Sleep(150 * time.Millisecond)
	}

	return nil
}

func sendTelegramMessage(bot *db.TelegramBot, text string) error {
	url := fmt.Sprintf("%s%s/sendMessage", telegramAPIURL, bot.Token)
	msg := telegramMessage{
		ChatID: bot.ChatID,
		Text:   text,
	}

	jsonMessage, err := json.Marshal(msg)
	if err != nil {
		logger.Error("Failed to marshal telegram message", "error", err)
		return err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonMessage))
	if err != nil {
		logger.Error("Failed to create telegram request", "error", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("Failed to send telegram message", "error", err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		logger.Error("Telegram API returned non-OK status", "status", resp.Status)
		return fmt.Errorf("telegram API returned status: %s", resp.Status)
	}
	logger.Debug("Telegram message sent successfully", "to", bot.ChatID)
	return nil
}

func sendBulkTelegram(n *notification) error {
	users, err := db.GetUsersWithTelegramBots()
	if err != nil {
		logger.Error("Failed to get users with telegram bots", "error", err)
		return err
	}

	for _, user := range users {
		n.UserID = user
		err := sendSingleTelegram(n)
		if err != nil {
			logger.Error("Failed to send telegram message to user", "userID", user, "error", err)
		}
	}
	return nil
}

func SendVMStatusUpdateNotification(userID uint, vmName string, status string) error {
	t := `Your VM "%s" status has changed to: %s

The status was not changed by you but by an external event.
If the status is "unknown" please contact an administrator.
`

	body := fmt.Sprintf(t, vmName, status)

	n := &notification{
		UserID:  userID,
		Subject: "VM Status Update",
		Body:    body,
	}
	err := n.save()
	if err != nil {
		logger.Error("Failed to save VM status update notification", "userID", userID, "error", err)
		return err
	}
	return nil
}

func SendGlobalSSHKeysChangeNotification() error {
	t := `The global SSH keys have been changed.
This means that if you had VMs with public keys included the VMs public key signature is changed.
At the next reboot you will probably get a warning from your SSH client about the host key being changed.
`

	n := &notification{
		UserID:  0, // 0 means all users
		Subject: "Global SSH Keys Changed",
		Body:    t,
	}
	err := n.save()
	if err != nil {
		logger.Error("Failed to save global SSH keys change notification", "error", err)
		return err
	}
	return nil
}

func SendVMExpirationNotification(userID uint, vmName string, daysLeft int) error {
	t := `Your VM "%s" is going to expire in less than %d days.
After the expiration date the VM will be deleted and all data will be lost.
To extend the lifetime of your VM please login and extend it.
`
	body := fmt.Sprintf(t, vmName, daysLeft)
	n := &notification{
		UserID:  userID,
		Subject: "VM Expiration Warning",
		Body:    body,
	}
	err := n.save()
	if err != nil {
		logger.Error("Failed to save VM expiration notification", "userID", userID, "error", err)
		return err
	}
	return nil
}

func SendVMEliminatedNotification(userID uint, vmName string) error {
	t := `Your VM "%s" has been deleted. Its lifetime has expired.
If you want to keep using our services please create a new VM.
`
	body := fmt.Sprintf(t, vmName)
	n := &notification{
		UserID:  userID,
		Subject: "VM Deleted",
		Body:    body,
	}
	err := n.save()
	if err != nil {
		logger.Error("Failed to save VM eliminated notification", "userID", userID, "error", err)
		return err
	}
	return nil
}

func SendVMStoppedNotification(userID uint, vmName string) error {
	t := `Your VM "%s" has been stopped lifetime expiration.
To use it again please login extend its lifetime.
`
	body := fmt.Sprintf(t, vmName)
	n := &notification{
		UserID:  userID,
		Subject: "VM Stopped",
		Body:    body,
	}
	err := n.save()
	if err != nil {
		logger.Error("Failed to save VM stopped notification", "userID", userID, "error", err)
		return err
	}
	return nil
}

func SendTestBotNotification(bot *db.TelegramBot, text string) error {
	err := sendTelegramMessage(bot, text)
	if err != nil {
		logger.Error("Failed to send test telegram message", "botID", bot.ID, "error", err)
	}
	return err
}
