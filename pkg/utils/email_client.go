package utils

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"mime"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"path/filepath"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/charset"
)

func init() {
	imap.CharsetReader = charset.Reader
}

// EmailClient provides functionality for sending and receiving emails
type EmailClient struct {
	smtpHost     string
	smtpPort     int
	imapHost     string
	imapPort     int
	username     string
	password     string
	smtpClient   *smtp.Client
	imapClient   *client.Client
	connected    bool
	lastActivity time.Time
}

// EmailMessage represents an email message
type EmailMessage struct {
	From        string            `json:"from"`
	To          []string          `json:"to"`
	Cc          []string          `json:"cc,omitempty"`
	Bcc         []string          `json:"bcc,omitempty"`
	Subject     string            `json:"subject"`
	Body        string            `json:"body"`
	HTML        string            `json:"html,omitempty"`
	Attachments []EmailAttachment `json:"attachments,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Date        time.Time         `json:"date,omitempty"`
	MessageID   string            `json:"message_id,omitempty"`
	Metadata    map[string]any    `json:"metadata,omitempty"`
}

// EmailAttachment represents an email attachment
type EmailAttachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Content     []byte `json:"content"`
}

// EmailFilter represents a filter for retrieving emails
type EmailFilter struct {
	Folder      string    `json:"folder"`
	Since       time.Time `json:"since,omitempty"`
	Before      time.Time `json:"before,omitempty"`
	From        string    `json:"from,omitempty"`
	To          string    `json:"to,omitempty"`
	Subject     string    `json:"subject,omitempty"`
	Unseen      bool      `json:"unseen,omitempty"`
	Limit       uint32    `json:"limit,omitempty"`
	MarkAsRead  bool      `json:"mark_as_read,omitempty"`
	WithBody    bool      `json:"with_body,omitempty"`
	BodyPreview uint32    `json:"body_preview,omitempty"`
}

// NewEmailClient creates a new email client
func NewEmailClient(smtpHost string, smtpPort int, imapHost string, imapPort int, username, password string) *EmailClient {
	return &EmailClient{
		smtpHost:     smtpHost,
		smtpPort:     smtpPort,
		imapHost:     imapHost,
		imapPort:     imapPort,
		username:     username,
		password:     password,
		lastActivity: time.Now(),
	}
}

// Connect connects to the SMTP and IMAP servers
func (c *EmailClient) Connect() error {
	// For Gmail, we'll use the smtp.SendMail function directly when sending
	// This is because Gmail requires TLS from the start
	// We'll set up a dummy client for now
	c.smtpClient = &smtp.Client{}

	// Connect to IMAP server
	imapAddr := fmt.Sprintf("%s:%d", c.imapHost, c.imapPort)
	imapClient, err := client.DialTLS(imapAddr, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to IMAP server: %w", err)
	}

	// Authenticate with IMAP server
	if err := imapClient.Login(c.username, c.password); err != nil {
		imapClient.Logout()
		return fmt.Errorf("IMAP authentication failed: %w", err)
	}

	c.imapClient = imapClient
	c.connected = true
	c.lastActivity = time.Now()

	return nil
}

// Close closes the connections to the SMTP and IMAP servers
func (c *EmailClient) Close() error {
	var smtpErr, imapErr error

	// Only try to close the SMTP client if it's not nil
	if c.smtpClient != nil {
		// Skip closing for our dummy SMTP client
		dummyClient := &smtp.Client{}
		if c.smtpClient != dummyClient {
			smtpErr = c.smtpClient.Close()
		}
		c.smtpClient = nil
	}

	// Only try to logout from IMAP if the client is not nil
	if c.imapClient != nil {
		// Use Logout() which is safer than Close()
		imapErr = c.imapClient.Logout()
		c.imapClient = nil
	}

	c.connected = false

	if smtpErr != nil {
		return smtpErr
	}
	return imapErr
}

// ensureConnected ensures that the client is connected
func (c *EmailClient) ensureConnected() error {
	if !c.connected || time.Since(c.lastActivity) > 5*time.Minute {
		if c.connected {
			c.Close()
		}
		return c.Connect()
	}
	return nil
}

// SendEmail sends an email
func (c *EmailClient) SendEmail(message EmailMessage) error {
	if err := c.ensureConnected(); err != nil {
		return err
	}

	// Create the message
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add headers
	fmt.Fprintf(&buf, "From: %s\r\n", message.From)
	fmt.Fprintf(&buf, "To: %s\r\n", strings.Join(message.To, ", "))
	if len(message.Cc) > 0 {
		fmt.Fprintf(&buf, "Cc: %s\r\n", strings.Join(message.Cc, ", "))
	}
	fmt.Fprintf(&buf, "Subject: %s\r\n", message.Subject)
	fmt.Fprintf(&buf, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&buf, "Content-Type: multipart/mixed; boundary=%s\r\n", writer.Boundary())
	fmt.Fprintf(&buf, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))

	// Add custom headers
	for key, value := range message.Headers {
		fmt.Fprintf(&buf, "%s: %s\r\n", key, value)
	}
	fmt.Fprintf(&buf, "\r\n")

	// Add text body
	if message.Body != "" {
		textPart, err := writer.CreatePart(textproto.MIMEHeader{
			"Content-Type":              {"text/plain; charset=UTF-8"},
			"Content-Transfer-Encoding": {"quoted-printable"},
		})
		if err != nil {
			return fmt.Errorf("failed to create text part: %w", err)
		}
		fmt.Fprintf(textPart, "%s", message.Body)
	}

	// Add HTML body
	if message.HTML != "" {
		htmlPart, err := writer.CreatePart(textproto.MIMEHeader{
			"Content-Type":              {"text/html; charset=UTF-8"},
			"Content-Transfer-Encoding": {"quoted-printable"},
		})
		if err != nil {
			return fmt.Errorf("failed to create HTML part: %w", err)
		}
		fmt.Fprintf(htmlPart, "%s", message.HTML)
	}

	// Add attachments
	for _, attachment := range message.Attachments {
		contentType := attachment.ContentType
		if contentType == "" {
			contentType = mime.TypeByExtension(filepath.Ext(attachment.Filename))
			if contentType == "" {
				contentType = "application/octet-stream"
			}
		}

		attachmentPart, err := writer.CreatePart(textproto.MIMEHeader{
			"Content-Type":              {fmt.Sprintf("%s; name=%q", contentType, attachment.Filename)},
			"Content-Disposition":       {fmt.Sprintf("attachment; filename=%q", attachment.Filename)},
			"Content-Transfer-Encoding": {"base64"},
		})
		if err != nil {
			return fmt.Errorf("failed to create attachment part: %w", err)
		}

		encoder := base64.NewEncoder(base64.StdEncoding, attachmentPart)
		encoder.Write(attachment.Content)
		encoder.Close()
	}

	// Close the writer
	writer.Close()

	// Create auth
	auth := smtp.PlainAuth("", c.username, c.password, c.smtpHost)

	// Create recipient list
	recipients := make([]string, 0, len(message.To)+len(message.Cc)+len(message.Bcc))
	recipients = append(recipients, message.To...)
	recipients = append(recipients, message.Cc...)
	recipients = append(recipients, message.Bcc...)

	// Send the email using smtp.SendMail
	smtpAddr := fmt.Sprintf("%s:%d", c.smtpHost, c.smtpPort)
	err := smtp.SendMail(smtpAddr, auth, message.From, recipients, buf.Bytes())
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	c.lastActivity = time.Now()
	return nil
}

// GetEmails retrieves emails based on the provided filter
func (c *EmailClient) GetEmails(filter EmailFilter) ([]EmailMessage, error) {
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	// Select the mailbox
	folder := filter.Folder
	if folder == "" {
		folder = "INBOX"
	}
	_, err := c.imapClient.Select(folder, !filter.MarkAsRead)
	if err != nil {
		return nil, fmt.Errorf("failed to select folder: %w", err)
	}

	// Create search criteria
	criteria := imap.NewSearchCriteria()

	// Add date criteria
	if !filter.Since.IsZero() {
		criteria.Since = filter.Since
	}
	if !filter.Before.IsZero() {
		criteria.Before = filter.Before
	}

	// Add text criteria
	if filter.From != "" {
		criteria.Header.Add("FROM", filter.From)
	}
	if filter.To != "" {
		criteria.Header.Add("TO", filter.To)
	}
	if filter.Subject != "" {
		criteria.Header.Add("SUBJECT", filter.Subject)
	}

	// Add flags criteria
	if filter.Unseen {
		criteria.WithoutFlags = []string{imap.SeenFlag}
	}

	// Search for messages
	uids, err := c.imapClient.Search(criteria)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	if len(uids) == 0 {
		return []EmailMessage{}, nil
	}

	// Limit the number of messages
	if filter.Limit > 0 && uint32(len(uids)) > filter.Limit {
		uids = uids[len(uids)-int(filter.Limit):]
	}

	// Create sequence set for fetching
	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uids...)

	// Define items to fetch
	var items []imap.FetchItem
	if filter.WithBody {
		items = []imap.FetchItem{imap.FetchEnvelope, imap.FetchFlags, imap.FetchInternalDate, imap.FetchRFC822Header}
		if filter.BodyPreview > 0 {
			items = append(items, imap.FetchBodyStructure)
		} else {
			items = append(items, imap.FetchRFC822)
		}
	} else {
		items = []imap.FetchItem{imap.FetchEnvelope, imap.FetchFlags, imap.FetchInternalDate}
	}

	// Fetch messages
	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	go func() {
		done <- c.imapClient.Fetch(seqSet, items, messages)
	}()

	// Process messages
	var emails []EmailMessage
	for msg := range messages {
		email := EmailMessage{
			Subject:   msg.Envelope.Subject,
			Date:      msg.Envelope.Date,
			MessageID: msg.Envelope.MessageId,
			Metadata: map[string]any{
				"uid":      msg.Uid,
				"flags":    msg.Flags,
				"date":     msg.InternalDate,
				"sequence": msg.SeqNum,
				"size":     msg.Size,
				"answered": hasFlag(msg.Flags, imap.AnsweredFlag),
				"deleted":  hasFlag(msg.Flags, imap.DeletedFlag),
				"draft":    hasFlag(msg.Flags, imap.DraftFlag),
				"flagged":  hasFlag(msg.Flags, imap.FlaggedFlag),
				"recent":   hasFlag(msg.Flags, imap.RecentFlag),
				"seen":     hasFlag(msg.Flags, imap.SeenFlag),
			},
		}

		// Process from address
		if len(msg.Envelope.From) > 0 {
			email.From = formatAddress(msg.Envelope.From[0])
		}

		// Process to addresses
		for _, addr := range msg.Envelope.To {
			email.To = append(email.To, formatAddress(addr))
		}

		// Process cc addresses
		for _, addr := range msg.Envelope.Cc {
			email.Cc = append(email.Cc, formatAddress(addr))
		}

		// Process body if requested
		if filter.WithBody && msg.Body != nil {
			// TODO: Process body parts
		}

		emails = append(emails, email)
	}

	// Check for errors
	if err := <-done; err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}

	c.lastActivity = time.Now()
	return emails, nil
}

// formatAddress formats an IMAP address
func formatAddress(addr *imap.Address) string {
	if addr.PersonalName != "" {
		return fmt.Sprintf("%s <%s@%s>", addr.PersonalName, addr.MailboxName, addr.HostName)
	}
	return fmt.Sprintf("%s@%s", addr.MailboxName, addr.HostName)
}

// hasFlag checks if a message has a specific flag
func hasFlag(flags []string, flag string) bool {
	for _, f := range flags {
		if f == flag {
			return true
		}
	}
	return false
}
