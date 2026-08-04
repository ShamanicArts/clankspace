package mailer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/smtp"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ShamanicArts/clankspace/internal/store"
)

type Sender interface {
	Send(context.Context, store.OutboxMessage) error
}

type SMTP struct {
	Addr, User, Password, From string
}

func (s SMTP) Send(_ context.Context, message store.OutboxMessage) error {
	if s.Addr == "" || s.From == "" {
		return errors.New("SMTP address and from address are required")
	}
	host := strings.Split(s.Addr, ":")[0]
	var auth smtp.Auth
	if s.User != "" {
		auth = smtp.PlainAuth("", s.User, s.Password, host)
	}
	body := "From: " + s.From + "\r\nTo: " + message.Recipient + "\r\nSubject: " + message.Subject + "\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" + strings.ReplaceAll(message.Body, "\n", "\r\n")
	return smtp.SendMail(s.Addr, auth, s.From, []string{message.Recipient}, []byte(body))
}

type File struct {
	Dir string
}

var unsafeFilename = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func (f File) Send(_ context.Context, message store.OutboxMessage) error {
	if f.Dir == "" {
		return errors.New("mail directory is required")
	}
	if err := os.MkdirAll(f.Dir, 0700); err != nil {
		return err
	}
	name := time.Now().UTC().Format("20060102T150405.000000000Z") + "-" + unsafeFilename.ReplaceAllString(message.Recipient, "_") + "-" + message.ID + ".json"
	payload := map[string]string{"id": message.ID, "to": message.Recipient, "template": message.Template, "subject": message.Subject, "body": message.Body}
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	return os.WriteFile(filepath.Join(f.Dir, name), body, 0600)
}

func Run(ctx context.Context, db *store.Store, sender Sender, interval time.Duration) error {
	if sender == nil {
		return nil
	}
	if interval <= 0 {
		interval = 2 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := deliver(ctx, db, sender); err != nil && !errors.Is(err, context.Canceled) {
			return fmt.Errorf("mail outbox: %w", err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func deliver(ctx context.Context, db *store.Store, sender Sender) error {
	messages, err := db.ClaimOutbox(ctx, 20)
	if err != nil {
		return err
	}
	for _, message := range messages {
		if err = sender.Send(ctx, message); err != nil {
			if markErr := db.MarkOutboxFailed(ctx, message.ID, err); markErr != nil {
				return markErr
			}
			continue
		}
		if err = db.MarkOutboxSent(ctx, message.ID); err != nil {
			return err
		}
	}
	return nil
}
