package notify

import (
	"context"
	"fmt"
	"log/slog"
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
