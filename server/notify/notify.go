package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/wneessen/go-mail"
	"samuelemusiani/sasso/server/config"
	"samuelemusiani/sasso/server/db"
)

var (
	logger *slog.Logger = nil
	email  string

	emailClient *mail.Client = nil

	workerContext    context.Context
	workerCancelFunc context.CancelFunc
	workerReturnChan chan error = make(chan error, 1)

	bucketLimiter24hInstance *bucketLimiter = nil
	bucketLimiter1mInstance  *bucketLimiter = nil
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

	// Send the notification via
	Mail     bool
	Telegram bool
}

func Init(l *slog.Logger, c config.Notifications) error {
	logger = l

	if !c.Enabled {
		slog.Info("Email notifications are disabled")

		return nil
	}

	if err := checkConfig(c); err != nil {
		return err
	}

	email = c.Email.Username

	var err error

	emailClient, err = mail.NewClient(c.Email.SMTPServer, mail.WithSMTPAuth(mail.SMTPAuthPlain), mail.WithUsername(c.Email.Username), mail.WithPassword(c.Email.Password), mail.WithSSL(), mail.WithPort(465))
	if err != nil {
		slog.Error("Failed to create mail client", "error", err)

		return err
	}

	if c.RateLimits {
		// We use two bucket limiters:
		// - the 24h limiter to limit the total number of emails sent per day
		// - the 1m limiter to limit bursts of emails
		bucketLimiter24hInstance = newBucketLimiter(float64(c.MaxPerDay)/(24*60*60), 1000) // 1000 notifications per day
		bucketLimiter1mInstance = newBucketLimiter(float64(c.MaxPerMinute)/60, 10)         // 20 notifications per minute, burst 10
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
	if workerCancelFunc == nil {
		// If notifications are disabled
		return nil
	}

	workerCancelFunc()

	var err error
	if workerReturnChan != nil {
		err = <-workerReturnChan
	}

	if err != nil && !errors.Is(err, context.Canceled) {
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

		if bucketLimiter1mInstance != nil && !bucketLimiter1mInstance.allow() {
			logger.Warn("Rate limit exceeded for the 1m bucket, skipping notifications", "userID", n.UserID)

			return
		}

		if bucketLimiter24hInstance != nil && !bucketLimiter24hInstance.allow() {
			logger.Warn("Rate limit exceeded for the 24h bucket, skipping notifications", "userID", n.UserID)

			return
		}

		if n.Email {
			err := sendEmail(&tmpN)
			if err != nil {
				logger.Error("Failed to send notification email", "userID", n.UserID, "error", err)
			}
		}

		if n.Telegram {
			err = sendTelegram(&tmpN)
			if err != nil {
				logger.Error("Failed to send notification telegram", "userID", n.UserID, "error", err)
			}
		}

		err = db.SetNotificationAsSent(n.ID)
		if err != nil {
			logger.Error("Failed to set notification as sent", "notificationID", n.ID, "error", err)
		}
	}
}

func (n *notification) save() error {
	return db.InsertNotification(n.UserID, n.Subject, n.Body, n.Mail, n.Telegram)
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
	bots, err := db.GetEnabledTelegramBotsByUserID(n.UserID)
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

func sendTelegramMessage(bot *db.TelegramBot, text string) (err error) {
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

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonMessage))
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

	defer func() {
		if e := resp.Body.Close(); e != nil {
			err = fmt.Errorf("error while closing telegram response body: %w", e)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		logger.Error("Telegram API returned non-OK status", "status", resp.Status)

		return fmt.Errorf("telegram API returned status: %s", resp.Status)
	}

	logger.Debug("Telegram message sent successfully", "to", bot.ChatID)

	return
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

func SendPortForwardNotificationToGroup(groupID uint, pf db.PortForward) error {
	members, err := db.GetUserIDsByGroupID(groupID)
	if err != nil {
		logger.Error("Failed to get group members for port forward notification", "groupID", groupID, "error", err)

		return err
	}

	for _, userID := range members {
		err := SendPortForwardNotification(userID, pf)
		if err != nil {
			logger.Error("Failed to send port forward notification to group member", "groupID", groupID, "userID", userID, "error", err)
		}
	}

	return nil
}

func SendPortForwardNotification(userID uint, pf db.PortForward) error {
	s, err := db.GetSettingsByUserID(userID)
	if err != nil {
		logger.Error("Failed to get user settings for port forward notification", "userID", userID, "error", err)

		return err
	}

	t := `Your port forwarding request has been %s!
Outside port: %d
Destination port: %d
Destination IP: %s
`

	body := fmt.Sprintf(t, map[bool]string{true: "approved", false: "disabled"}[pf.Approved], pf.OutPort, pf.DestPort, pf.DestIP)

	n := &notification{
		UserID:   userID,
		Subject:  "Port Forwarding Status Update",
		Mail:     s.MailPortForwardNotification,
		Telegram: s.TelegramPortForwardNotification,
		Body:     body,
	}

	err = n.save()
	if err != nil {
		logger.Error("Failed to save port forward notification", "userID", userID, "error", err)

		return err
	}

	return nil
}

func SendVMStatusUpdateNotificationToGroup(groupID uint, vmName string, status string) error {
	members, err := db.GetUserIDsByGroupID(groupID)
	if err != nil {
		logger.Error("Failed to get group members for VM status update notification", "groupID", groupID, "error", err)

		return err
	}

	for _, userID := range members {
		err := SendVMStatusUpdateNotification(userID, vmName, status)
		if err != nil {
			logger.Error("Failed to send VM status update notification to group member", "groupID", groupID, "userID", userID, "error", err)
		}
	}

	return nil
}

func SendVMStatusUpdateNotification(userID uint, vmName string, status string) error {
	s, err := db.GetSettingsByUserID(userID)
	if err != nil {
		logger.Error("Failed to get user settings for VM status update notification", "userID", userID, "error", err)

		return err
	}

	t := `Your VM "%s" status has changed to: %s

The status was not changed by you but by an external event.
If the status is "unknown" please contact an administrator.
`

	body := fmt.Sprintf(t, vmName, status)

	n := &notification{
		UserID:   userID,
		Subject:  "VM Status Update",
		Mail:     s.MailVMStatusUpdateNotification,
		Telegram: s.TelegramVMStatusUpdateNotification,
		Body:     body,
	}

	err = n.save()
	if err != nil {
		logger.Error("Failed to save VM status update notification", "userID", userID, "error", err)

		return err
	}

	return nil
}

func SendGlobalSSHKeysChangeNotification() error {
	s, err := db.GetSettingsByUserID(0)
	if err != nil {
		logger.Error("Failed to get user settings for global SSH keys change notification", "userID", 0, "error", err)

		return err
	}

	t := `The global SSH keys have been changed.
This means that if you had VMs with public keys included the VMs public key signature is changed.
At the next reboot you will probably get a warning from your SSH client about the host key being changed.
`

	n := &notification{
		UserID:   0, // 0 means all users
		Subject:  "Global SSH Keys Changed",
		Mail:     s.MailGlobalSSHKeysChangeNotification,
		Telegram: s.TelegramGlobalSSHKeysChangeNotification,
		Body:     t,
	}

	err = n.save()
	if err != nil {
		logger.Error("Failed to save global SSH keys change notification", "error", err)

		return err
	}

	return nil
}

func SendVMExpirationNotificationToGroup(groupID uint, vmName string, daysLeft int) error {
	members, err := db.GetUserIDsByGroupID(groupID)
	if err != nil {
		logger.Error("Failed to get group members for VM expiration notification", "groupID", groupID, "error", err)

		return err
	}

	for _, userID := range members {
		err := SendVMExpirationNotification(userID, vmName, daysLeft)
		if err != nil {
			logger.Error("Failed to send VM expiration notification to group member", "groupID", groupID, "userID", userID, "error", err)
		}
	}

	return nil
}

func SendVMExpirationNotification(userID uint, vmName string, daysLeft int) error {
	s, err := db.GetSettingsByUserID(userID)
	if err != nil {
		logger.Error("Failed to get user settings for VM expiration notification", "userID", userID, "error", err)

		return err
	}

	t := `Your VM "%s" is going to expire in less than %d days.
After the expiration date the VM will be deleted and all data will be lost.
To extend the lifetime of your VM please login and extend it.
`
	body := fmt.Sprintf(t, vmName, daysLeft)
	n := &notification{
		UserID:   userID,
		Subject:  "VM Expiration Warning",
		Mail:     s.MailVMExpirationNotification,
		Telegram: s.TelegramVMExpirationNotification,
		Body:     body,
	}

	err = n.save()
	if err != nil {
		logger.Error("Failed to save VM expiration notification", "userID", userID, "error", err)

		return err
	}

	return nil
}

func SendVMEliminatedNotificationToGroup(groupID uint, vmName string) error {
	members, err := db.GetUserIDsByGroupID(groupID)
	if err != nil {
		logger.Error("Failed to get group members for VM eliminated notification", "groupID", groupID, "error", err)

		return err
	}

	for _, userID := range members {
		err := SendVMEliminatedNotification(userID, vmName)
		if err != nil {
			logger.Error("Failed to send VM eliminated notification to group member", "groupID", groupID, "userID", userID, "error", err)
		}
	}

	return nil
}

func SendVMEliminatedNotification(userID uint, vmName string) error {
	s, err := db.GetSettingsByUserID(userID)
	if err != nil {
		logger.Error("Failed to get user settings for VM eliminated notification", "userID", userID, "error", err)

		return err
	}

	t := `Your VM "%s" has been deleted. Its lifetime has expired.
If you want to keep using our services please create a new VM.
`
	body := fmt.Sprintf(t, vmName)
	n := &notification{
		UserID:   userID,
		Subject:  "VM Deleted",
		Mail:     s.MailVMEliminatedNotification,
		Telegram: s.TelegramVMEliminatedNotification,
		Body:     body,
	}

	err = n.save()
	if err != nil {
		logger.Error("Failed to save VM eliminated notification", "userID", userID, "error", err)

		return err
	}

	return nil
}

func SendVMStoppedNotificationToGroup(groupID uint, vmName string) error {
	members, err := db.GetUserIDsByGroupID(groupID)
	if err != nil {
		logger.Error("Failed to get group members for VM stopped notification", "groupID", groupID, "error", err)

		return err
	}

	for _, userID := range members {
		err := SendVMStoppedNotification(userID, vmName)
		if err != nil {
			logger.Error("Failed to send VM stopped notification to group member", "groupID", groupID, "userID", userID, "error", err)
		}
	}

	return nil
}

func SendVMStoppedNotification(userID uint, vmName string) error {
	s, err := db.GetSettingsByUserID(userID)
	if err != nil {
		logger.Error("Failed to get user settings for VM stopped notification", "userID", userID, "error", err)

		return err
	}

	t := `Your VM "%s" has been stopped because of lifetime expiration.
To use it again please login and extend its lifetime.
`
	body := fmt.Sprintf(t, vmName)
	n := &notification{
		UserID:   userID,
		Subject:  "VM Stopped",
		Mail:     s.MailVMStoppedNotification,
		Telegram: s.TelegramVMStoppedNotification,
		Body:     body,
	}

	err = n.save()
	if err != nil {
		logger.Error("Failed to save VM stopped notification", "userID", userID, "error", err)

		return err
	}

	return nil
}

func SendLifetimeOfVMExpiredToGroup(groupID uint, vmName string) error {
	members, err := db.GetUserIDsByGroupID(groupID)
	if err != nil {
		logger.Error("Failed to get group members for lifetime of VM expired notification", "groupID", groupID, "error", err)

		return err
	}

	for _, userID := range members {
		err := SendLifetimeOfVMExpired(userID, vmName)
		if err != nil {
			logger.Error("Failed to send lifetime of VM expired notification to group member", "groupID", groupID, "userID", userID, "error", err)
		}
	}

	return nil
}

func SendLifetimeOfVMExpired(userID uint, vmName string) error {
	s, err := db.GetSettingsByUserID(userID)
	if err != nil {
		logger.Error("Failed to get user settings for lifetime of VM expired notification", "userID", userID, "error", err)

		return err
	}

	t := `The lifetime of your VM "%s" has expired.
To use it again please login and extend its lifetime.
`
	body := fmt.Sprintf(t, vmName)
	n := &notification{
		UserID:   userID,
		Subject:  "VM Lifetime Expired",
		Mail:     s.MailLifetimeOfVMExpiredNotification,
		Telegram: s.TelegramLifetimeOfVMExpiredNotification,
		Body:     body,
	}

	err = n.save()
	if err != nil {
		logger.Error("Failed to save lifetime of VM expired notification", "userID", userID, "error", err)

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

func SendSSHKeysChangedOnVMToGroup(groupID uint, vmName string) error {
	members, err := db.GetUserIDsByGroupID(groupID)
	if err != nil {
		logger.Error("Failed to get group members for SSH keys changed notification", "groupID", groupID, "error", err)

		return err
	}

	for _, userID := range members {
		err := SendSSHKeysChangedOnVM(userID, vmName)
		if err != nil {
			logger.Error("Failed to send SSH keys changed notification to group member", "groupID", groupID, "userID", userID, "error", err)
		}
	}

	return nil
}

func SendSSHKeysChangedOnVM(userID uint, vmName string) error {
	s, err := db.GetSettingsByUserID(userID)
	if err != nil {
		logger.Error("Failed to get user settings for SSH keys changed notification", "userID", userID, "error", err)

		return err
	}

	t := `The SSH keys on the VM "%s" have been changed.
If this is a group VM it often means that one of the group members has changed
his SSH keys. On the next reboot you will probably get a warning from your SSH
client about the host key being changed.
`
	body := fmt.Sprintf(t, vmName)
	n := &notification{
		UserID:   userID,
		Subject:  "VM SSH Keys Changed",
		Mail:     s.MailSSHKeysChangedOnVMNotification,
		Telegram: s.TelegramSSHKeysChangedOnVMNotification,
		Body:     body,
	}

	err = n.save()
	if err != nil {
		logger.Error("Failed to save SSH keys changed notification", "userID", userID, "error", err)

		return err
	}

	return nil
}

func SendUserInvitation(userID uint, groupName, role string) error {
	s, err := db.GetSettingsByUserID(userID)
	if err != nil {
		logger.Error("Failed to get user settings for user invitation notification", "userID", userID, "error", err)

		return err
	}

	t := `You have been invited to join the group "%s" with the role of "%s".
To accept the invitation please login to your account and navigate to the
groups section.
`
	body := fmt.Sprintf(t, groupName, role)
	n := &notification{
		UserID:   userID,
		Subject:  "Group Invitation",
		Mail:     s.MailUserInvitationNotification,
		Telegram: s.TelegramUserInvitationNotification,
		Body:     body,
	}

	err = n.save()
	if err != nil {
		logger.Error("Failed to save user invitation notification", "userID", userID, "error", err)

		return err
	}

	return nil
}

func SendUserRemovalFromGroupNotification(userID uint, groupName string) error {
	s, err := db.GetSettingsByUserID(userID)
	if err != nil {
		logger.Error("Failed to get user settings for user removal from group notification", "userID", userID, "error", err)

		return err
	}

	t := `You have been removed from the group "%s".`
	body := fmt.Sprintf(t, groupName)
	n := &notification{
		UserID:   userID,
		Subject:  "Removed from Group",
		Mail:     s.MailUserRemovalFromGroupNotification,
		Telegram: s.TelegramUserRemovalFromGroupNotification,
		Body:     body,
	}

	err = n.save()
	if err != nil {
		logger.Error("Failed to save user removal from group notification", "userID", userID, "error", err)

		return err
	}

	return nil
}

func checkConfig(c config.Notifications) error {
	if !c.Enabled {
		return nil
	}

	if c.RateLimits {
		if c.MaxPerDay <= 0 {
			return fmt.Errorf("notifications max per day must be greater than 0 when rate limits are enabled")
		}

		if c.MaxPerMinute <= 0 {
			return fmt.Errorf("notifications max per minute must be greater than 0 when rate limits are enabled")
		}
	}

	if c.Email.Enabled {
		if c.Email.SMTPServer == "" {
			return fmt.Errorf("notifications SMTP server is empty")
		}

		if c.Email.Username == "" {
			return fmt.Errorf("notifications SMTP username is empty")
		}

		if c.Email.Password == "" {
			return fmt.Errorf("notifications SMTP password is empty")
		}
	}

	return nil
}
