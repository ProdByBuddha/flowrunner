# Email Nodes Documentation

Flowrunner provides two email-related nodes for sending and receiving emails: `email.send` (SMTP) and `email.receive` (IMAP). These nodes allow workflows to interact with email systems for notifications, data collection, and automation.

## SMTP Node (email.send)

The SMTP node allows sending emails with support for plain text and HTML content, attachments, and custom headers.

### Basic Usage

```yaml
smtp_node:
  type: "email.send"
  params:
    smtp_host: "smtp.gmail.com"
    smtp_port: 587
    username: "${secrets.EMAIL_USERNAME}"
    password: "${secrets.EMAIL_PASSWORD}"
    from: "sender@example.com"
    to: "recipient@example.com"
    subject: "Hello from Flowrunner"
    body: "This is a test email sent from the Flowrunner SMTP node."
```

### With HTML Content

```yaml
html_email_node:
  type: "email.send"
  params:
    smtp_host: "smtp.gmail.com"
    smtp_port: 587
    username: "${secrets.EMAIL_USERNAME}"
    password: "${secrets.EMAIL_PASSWORD}"
    from: "sender@example.com"
    to: "recipient@example.com"
    subject: "HTML Email from Flowrunner"
    body: "This is a test email with HTML content."
    html: "<h1>Hello</h1><p>This is a <b>test email</b> with HTML content.</p>"
```

### With Multiple Recipients and CC/BCC

```yaml
multi_recipient_node:
  type: "email.send"
  params:
    smtp_host: "smtp.gmail.com"
    smtp_port: 587
    username: "${secrets.EMAIL_USERNAME}"
    password: "${secrets.EMAIL_PASSWORD}"
    from: "sender@example.com"
    to: ["recipient1@example.com", "recipient2@example.com"]
    cc: ["cc1@example.com", "cc2@example.com"]
    bcc: ["bcc@example.com"]
    subject: "Email with Multiple Recipients"
    body: "This email is sent to multiple recipients."
```

### With Attachments

```yaml
attachment_node:
  type: "email.send"
  params:
    smtp_host: "smtp.gmail.com"
    smtp_port: 587
    username: "${secrets.EMAIL_USERNAME}"
    password: "${secrets.EMAIL_PASSWORD}"
    from: "sender@example.com"
    to: "recipient@example.com"
    subject: "Email with Attachment"
    body: "Please find the attached document."
    attachments:
      - filename: "document.pdf"
        content_type: "application/pdf"
        content: "${base64_encoded_content}"
      - filename: "image.jpg"
        content_type: "image/jpeg"
        content: "${base64_encoded_image}"
```

### With Custom Headers

```yaml
headers_node:
  type: "email.send"
  params:
    smtp_host: "smtp.gmail.com"
    smtp_port: 587
    username: "${secrets.EMAIL_USERNAME}"
    password: "${secrets.EMAIL_PASSWORD}"
    from: "sender@example.com"
    to: "recipient@example.com"
    subject: "Email with Custom Headers"
    body: "This email has custom headers."
    headers:
      X-Priority: "1"
      X-Custom-Header: "Custom Value"
```

## IMAP Node (email.receive)

The IMAP node allows retrieving emails with filtering options and full content access.

### Basic Usage

```yaml
imap_node:
  type: "email.receive"
  params:
    imap_host: "imap.gmail.com"
    imap_port: 993
    username: "${secrets.EMAIL_USERNAME}"
    password: "${secrets.EMAIL_PASSWORD}"
    folder: "INBOX"
    limit: 10
```

### With Filtering

```yaml
filtered_imap_node:
  type: "email.receive"
  params:
    imap_host: "imap.gmail.com"
    imap_port: 993
    username: "${secrets.EMAIL_USERNAME}"
    password: "${secrets.EMAIL_PASSWORD}"
    folder: "INBOX"
    limit: 10
    unseen: true
    from: "important@example.com"
    subject: "Urgent"
    since: "2023-01-01T00:00:00Z"
```

### With Body Content

```yaml
body_imap_node:
  type: "email.receive"
  params:
    imap_host: "imap.gmail.com"
    imap_port: 993
    username: "${secrets.EMAIL_USERNAME}"
    password: "${secrets.EMAIL_PASSWORD}"
    folder: "INBOX"
    limit: 5
    with_body: true
    mark_as_read: false
```

## Parameters

### SMTP Node Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `smtp_host` | string | Yes | SMTP server hostname |
| `smtp_port` | number | No | SMTP server port (default: 587) |
| `username` | string | Yes | Email account username |
| `password` | string | Yes | Email account password |
| `from` | string | No | Sender email address (defaults to username) |
| `to` | string/array | Yes | Recipient email address(es) |
| `cc` | string/array | No | CC recipient email address(es) |
| `bcc` | string/array | No | BCC recipient email address(es) |
| `subject` | string | Yes | Email subject |
| `body` | string | No | Plain text email body |
| `html` | string | No | HTML email body |
| `attachments` | array | No | Email attachments |
| `headers` | object | No | Custom email headers |

### IMAP Node Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `imap_host` | string | Yes | IMAP server hostname |
| `imap_port` | number | No | IMAP server port (default: 993) |
| `username` | string | Yes | Email account username |
| `password` | string | Yes | Email account password |
| `folder` | string | No | Mailbox folder to search (default: "INBOX") |
| `limit` | number | No | Maximum number of emails to retrieve |
| `unseen` | boolean | No | Only retrieve unread emails |
| `from` | string | No | Filter by sender |
| `to` | string | No | Filter by recipient |
| `subject` | string | No | Filter by subject |
| `since` | string | No | Filter by date (RFC3339 format) |
| `before` | string | No | Filter by date (RFC3339 format) |
| `with_body` | boolean | No | Retrieve email body content |
| `mark_as_read` | boolean | No | Mark retrieved emails as read |
| `body_preview` | number | No | Maximum length of body preview |

## Output

### SMTP Node Output

```json
{
  "status": "sent",
  "from": "sender@example.com",
  "to": ["recipient@example.com"],
  "cc": ["cc@example.com"],
  "bcc": ["bcc@example.com"],
  "subject": "Email Subject"
}
```

### IMAP Node Output

```json
[
  {
    "subject": "Email Subject",
    "from": "sender@example.com",
    "to": ["recipient@example.com"],
    "cc": ["cc@example.com"],
    "date": "2023-01-01T12:00:00Z",
    "body": "Email body content...",
    "html": "<html><body>HTML content...</body></html>",
    "headers": {
      "Message-ID": "<message-id>",
      "Content-Type": "multipart/alternative; boundary=boundary"
    },
    "messageId": "<message-id>",
    "metadata": {
      "uid": 123,
      "flags": ["\\Seen"],
      "date": "2023-01-01T12:00:00Z",
      "sequence": 1,
      "size": 1024,
      "answered": false,
      "deleted": false,
      "draft": false,
      "flagged": false,
      "recent": true,
      "seen": true
    }
  }
]
```

## Error Handling

The email nodes handle various error scenarios:

- Connection errors
- Authentication failures
- Invalid recipient addresses
- Server timeouts
- Message parsing errors

Errors are propagated through the flow execution and can be handled by error paths in the flow definition.

## Examples

### Send Notification Email

```yaml
notification_node:
  type: "email.send"
  params:
    smtp_host: "smtp.gmail.com"
    smtp_port: 587
    username: "${secrets.EMAIL_USERNAME}"
    password: "${secrets.EMAIL_PASSWORD}"
    from: "alerts@example.com"
    to: "admin@example.com"
    subject: "System Alert: ${alert.title}"
    body: "Alert Details:\n\nTime: ${alert.time}\nSeverity: ${alert.severity}\nMessage: ${alert.message}"
    html: "<h1>System Alert: ${alert.title}</h1><p><b>Time:</b> ${alert.time}<br><b>Severity:</b> ${alert.severity}<br><b>Message:</b> ${alert.message}</p>"
```

### Process Incoming Support Emails

```yaml
support_email_node:
  type: "email.receive"
  params:
    imap_host: "imap.gmail.com"
    imap_port: 993
    username: "${secrets.SUPPORT_EMAIL}"
    password: "${secrets.SUPPORT_PASSWORD}"
    folder: "INBOX"
    limit: 20
    unseen: true
    with_body: true
    mark_as_read: true
```

### Email Autoresponder

```yaml
# First node: Check for new emails
check_emails:
  type: "email.receive"
  params:
    imap_host: "imap.gmail.com"
    imap_port: 993
    username: "${secrets.EMAIL_USERNAME}"
    password: "${secrets.EMAIL_PASSWORD}"
    folder: "INBOX"
    limit: 10
    unseen: true
    with_body: true
  next:
    default: "send_response"

# Second node: Send autoresponse
send_response:
  type: "email.send"
  params:
    smtp_host: "smtp.gmail.com"
    smtp_port: 587
    username: "${secrets.EMAIL_USERNAME}"
    password: "${secrets.EMAIL_PASSWORD}"
    from: "${secrets.EMAIL_USERNAME}"
    to: "${result[0].from}"
    subject: "Re: ${result[0].subject}"
    body: "Thank you for your email. This is an automated response. We will get back to you shortly."
    html: "<h2>Thank you for your email</h2><p>This is an automated response. We will get back to you shortly.</p>"
```