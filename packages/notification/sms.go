package notification

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// SmsMessage is an SMS payload.
type SmsMessage struct {
	To   string
	Body string
	From string
	Meta map[string]any
}

// SmsNotification optionally supplies an SMS representation.
type SmsNotification interface {
	ToSms(notifiable Notifiable) *SmsMessage
}

// SmsSender delivers SMS messages.
type SmsSender interface {
	Send(message *SmsMessage) error
}

// MemorySmsSender records SMS deliveries for tests and demos.
type MemorySmsSender struct {
	mu      sync.Mutex
	Entries []*SmsMessage
}

// Send records the SMS.
func (s *MemorySmsSender) Send(message *SmsMessage) error {
	if message == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *message
	s.Entries = append(s.Entries, &cp)
	return nil
}

// Last returns the most recent SMS.
func (s *MemorySmsSender) Last() (*SmsMessage, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.Entries) == 0 {
		return nil, false
	}
	return s.Entries[len(s.Entries)-1], true
}

// LogSmsSender writes SMS payloads to a logger-style sink.
type LogSmsSender struct {
	Log func(format string, args ...any)
}

// Send logs the SMS.
func (s *LogSmsSender) Send(message *SmsMessage) error {
	if message == nil {
		return nil
	}
	log := s.Log
	if log == nil {
		log = func(format string, args ...any) { fmt.Printf(format+"\n", args...) }
	}
	log("sms to=%s from=%s body=%s", message.To, message.From, message.Body)
	return nil
}

// HTTPSmsSender POSTs SMS payloads to a generic HTTP endpoint.
// Default: POST JSON {to,body,from} with Authorization: Bearer {Token}.
// Set Form=true to send application/x-www-form-urlencoded instead.
type HTTPSmsSender struct {
	Endpoint   string
	Method     string
	Token      string
	AuthHeader string // full Authorization header value; overrides Token when set
	Form       bool
	Client     *http.Client
}

// Send delivers the SMS via HTTP.
func (s *HTTPSmsSender) Send(message *SmsMessage) error {
	if message == nil {
		return nil
	}
	if strings.TrimSpace(s.Endpoint) == "" {
		return fmt.Errorf("notification: SMS_URL / HTTPSmsSender.Endpoint is required")
	}
	method := strings.ToUpper(strings.TrimSpace(s.Method))
	if method == "" {
		method = http.MethodPost
	}
	client := s.Client
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}

	var body io.Reader
	contentType := "application/json"
	if s.Form {
		contentType = "application/x-www-form-urlencoded"
		values := url.Values{}
		values.Set("To", message.To)
		values.Set("Body", message.Body)
		values.Set("From", message.From)
		body = strings.NewReader(values.Encode())
	} else {
		payload, err := json.Marshal(map[string]string{
			"to":   message.To,
			"body": message.Body,
			"from": message.From,
		})
		if err != nil {
			return err
		}
		body = strings.NewReader(string(payload))
	}

	req, err := http.NewRequest(method, s.Endpoint, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json")
	if s.AuthHeader != "" {
		req.Header.Set("Authorization", s.AuthHeader)
	} else if s.Token != "" {
		req.Header.Set("Authorization", "Bearer "+s.Token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(raw))
		if msg == "" {
			msg = resp.Status
		}
		return fmt.Errorf("notification: sms http %d: %s", resp.StatusCode, msg)
	}
	return nil
}

// TwilioSmsSender delivers SMS via the Twilio REST API.
type TwilioSmsSender struct {
	AccountSID string
	AuthToken  string
	From       string
	Client     *http.Client
}

// Send delivers the SMS through Twilio.
func (s *TwilioSmsSender) Send(message *SmsMessage) error {
	if message == nil {
		return nil
	}
	sid := strings.TrimSpace(s.AccountSID)
	token := strings.TrimSpace(s.AuthToken)
	if sid == "" || token == "" {
		return fmt.Errorf("notification: TWILIO_ACCOUNT_SID and TWILIO_AUTH_TOKEN are required")
	}
	from := message.From
	if from == "" {
		from = s.From
	}
	if from == "" {
		return fmt.Errorf("notification: twilio From is required")
	}
	endpoint := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", sid)
	values := url.Values{}
	values.Set("To", message.To)
	values.Set("From", from)
	values.Set("Body", message.Body)

	client := s.Client
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(sid, token)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(raw))
		if msg == "" {
			msg = resp.Status
		}
		return fmt.Errorf("notification: twilio sms %d: %s", resp.StatusCode, msg)
	}
	return nil
}

// SmsChannel sends notifications via SMS.
type SmsChannel struct {
	sender SmsSender
	from   string
}

// NewSmsChannel creates an SMS notification channel.
func NewSmsChannel(sender SmsSender, from ...string) *SmsChannel {
	if sender == nil {
		sender = &MemorySmsSender{}
	}
	ch := &SmsChannel{sender: sender}
	if len(from) > 0 {
		ch.from = from[0]
	}
	return ch
}

// Send delivers the SMS representation.
func (c *SmsChannel) Send(notifiable Notifiable, notification Notification) error {
	var message *SmsMessage
	if n, ok := notification.(SmsNotification); ok {
		message = n.ToSms(notifiable)
	}
	if message == nil {
		return nil
	}
	if message.To == "" {
		message.To = notifiable.RouteNotificationFor("sms")
	}
	if message.To == "" {
		return fmt.Errorf("notification: sms recipient is empty")
	}
	if message.From == "" {
		message.From = c.from
	}
	return c.sender.Send(message)
}
