package notification

import (
	"fmt"
	"strings"
	"text/template"
)

// TemplateEngineImpl implements TemplateEngine interface
type TemplateEngineImpl struct {
	templates map[string]string
}

// NewTemplateEngine creates a new template engine
func NewTemplateEngine() TemplateEngine {
	return &TemplateEngineImpl{
		templates: make(map[string]string),
	}
}

// ProcessTemplate processes a template with data
func (t *TemplateEngineImpl) ProcessTemplate(templateStr string, data map[string]interface{}) (string, error) {
	tmpl, err := template.New("notification").Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// ProcessEmailTemplate processes an email template
func (t *TemplateEngineImpl) ProcessEmailTemplate(templateID string, data map[string]interface{}) (subject, body string, err error) {
	// Get template by ID
	templateStr, exists := t.templates[templateID]
	if !exists {
		// Use default template
		templateStr = t.getDefaultEmailTemplate(templateID)
	}

	// Process template
	processed, err := t.ProcessTemplate(templateStr, data)
	if err != nil {
		return "", "", err
	}

	// Split subject and body
	parts := strings.SplitN(processed, "\n---\n", 2)
	if len(parts) == 2 {
		subject = strings.TrimSpace(parts[0])
		body = strings.TrimSpace(parts[1])
	} else {
		subject = "Notification"
		body = processed
	}

	return subject, body, nil
}

// ProcessSMSTemplate processes an SMS template
func (t *TemplateEngineImpl) ProcessSMSTemplate(templateID string, data map[string]interface{}) (string, error) {
	// Get template by ID
	templateStr, exists := t.templates[templateID]
	if !exists {
		// Use default template
		templateStr = t.getDefaultSMSTemplate(templateID)
	}

	return t.ProcessTemplate(templateStr, data)
}

// ProcessTelegramTemplate processes a Telegram template
func (t *TemplateEngineImpl) ProcessTelegramTemplate(templateID string, data map[string]interface{}) (string, error) {
	// Get template by ID
	templateStr, exists := t.templates[templateID]
	if !exists {
		// Use default template
		templateStr = t.getDefaultTelegramTemplate(templateID)
	}

	return t.ProcessTemplate(templateStr, data)
}

// getDefaultEmailTemplate returns a default email template for the given type
func (t *TemplateEngineImpl) getDefaultEmailTemplate(templateID string) string {
	switch templateID {
	case string(NotificationTypeConversionStarted):
		return `{{.notification.title}}
---
<html>
<body>
<h2>{{.notification.title}}</h2>
<p>{{.notification.message}}</p>
<p>Conversion ID: {{.notification.data.conversionId}}</p>
<p>Status: {{.notification.data.status}}</p>
</body>
</html>`

	case string(NotificationTypeConversionCompleted):
		return `{{.notification.title}}
---
<html>
<body>
<h2>{{.notification.title}}</h2>
<p>{{.notification.message}}</p>
<p>Conversion ID: {{.notification.data.conversionId}}</p>
<p>Result Image ID: {{.notification.data.resultImageId}}</p>
<p>Status: {{.notification.data.status}}</p>
</body>
</html>`

	case string(NotificationTypeConversionFailed):
		return `{{.notification.title}}
---
<html>
<body>
<h2>{{.notification.title}}</h2>
<p>{{.notification.message}}</p>
<p>Conversion ID: {{.notification.data.conversionId}}</p>
<p>Error: {{.notification.data.errorMessage}}</p>
<p>Status: {{.notification.data.status}}</p>
</body>
</html>`

	case string(NotificationTypeQuotaExhausted):
		return `{{.notification.title}}
---
<html>
<body>
<h2>{{.notification.title}}</h2>
<p>{{.notification.message}}</p>
<p>Quota Type: {{.notification.data.quotaType}}</p>
<p><a href="/upgrade">Upgrade your plan</a></p>
</body>
</html>`

	case string(NotificationTypeQuotaWarning):
		return `{{.notification.title}}
---
<html>
<body>
<h2>{{.notification.title}}</h2>
<p>{{.notification.message}}</p>
<p>Remaining: {{.notification.data.remaining}}</p>
<p>Quota Type: {{.notification.data.quotaType}}</p>
</body>
</html>`

	case string(NotificationTypePaymentSuccess):
		return `{{.notification.title}}
---
<html>
<body>
<h2>{{.notification.title}}</h2>
<p>{{.notification.message}}</p>
<p>Plan: {{.notification.data.planName}}</p>
<p>Payment ID: {{.notification.data.paymentId}}</p>
</body>
</html>`

	case string(NotificationTypePaymentFailed):
		return `{{.notification.title}}
---
<html>
<body>
<h2>{{.notification.title}}</h2>
<p>{{.notification.message}}</p>
<p>Reason: {{.notification.data.reason}}</p>
<p>Payment ID: {{.notification.data.paymentId}}</p>
</body>
</html>`

	case string(NotificationTypeCriticalError):
		return `{{.notification.title}}
---
<html>
<body>
<h2>{{.notification.title}}</h2>
<p>{{.notification.message}}</p>
<p>Error Type: {{.notification.data.errorType}}</p>
<p>Timestamp: {{.notification.data.timestamp}}</p>
</body>
</html>`

	default:
		return `{{.notification.title}}
---
<html>
<body>
<h2>{{.notification.title}}</h2>
<p>{{.notification.message}}</p>
</body>
</html>`
	}
}

// getDefaultSMSTemplate returns a default SMS template for the given type
func (t *TemplateEngineImpl) getDefaultSMSTemplate(templateID string) string {
	switch templateID {
	case string(NotificationTypeConversionStarted):
		return `{{.notification.title}}: {{.notification.message}} (ID: {{.notification.data.conversionId}})`

	case string(NotificationTypeConversionCompleted):
		return `{{.notification.title}}: {{.notification.message}} (ID: {{.notification.data.conversionId}})`

	case string(NotificationTypeConversionFailed):
		return `{{.notification.title}}: {{.notification.message}} (ID: {{.notification.data.conversionId}})`

	case string(NotificationTypeQuotaExhausted):
		return `{{.notification.title}}: {{.notification.message}}`

	case string(NotificationTypeQuotaWarning):
		return `{{.notification.title}}: {{.notification.message}} ({{.notification.data.remaining}} remaining)`

	case string(NotificationTypePaymentSuccess):
		return `{{.notification.title}}: {{.notification.message}} ({{.notification.data.planName}})`

	case string(NotificationTypePaymentFailed):
		return `{{.notification.title}}: {{.notification.message}}`

	default:
		return `{{.notification.title}}: {{.notification.message}}`
	}
}

// getDefaultTelegramTemplate returns a default Telegram template for the given type
func (t *TemplateEngineImpl) getDefaultTelegramTemplate(templateID string) string {
	switch templateID {
	case string(NotificationTypeConversionStarted):
		return `*{{.notification.title}}*

{{.notification.message}}

*Conversion ID:* {{.notification.data.conversionId}}
*Status:* {{.notification.data.status}}`

	case string(NotificationTypeConversionCompleted):
		return `*{{.notification.title}}* ✅

{{.notification.message}}

*Conversion ID:* {{.notification.data.conversionId}}
*Result Image ID:* {{.notification.data.resultImageId}}
*Status:* {{.notification.data.status}}`

	case string(NotificationTypeConversionFailed):
		return `*{{.notification.title}}* ❌

{{.notification.message}}

*Conversion ID:* {{.notification.data.conversionId}}
*Error:* {{.notification.data.errorMessage}}
*Status:* {{.notification.data.status}}`

	case string(NotificationTypeQuotaExhausted):
		return `*{{.notification.title}}* ⚠️

{{.notification.message}}

*Quota Type:* {{.notification.data.quotaType}}

[Upgrade your plan](/upgrade)`

	case string(NotificationTypeQuotaWarning):
		return `*{{.notification.title}}* ⚠️

{{.notification.message}}

*Remaining:* {{.notification.data.remaining}}
*Quota Type:* {{.notification.data.quotaType}}`

	case string(NotificationTypePaymentSuccess):
		return `*{{.notification.title}}* ✅

{{.notification.message}}

*Plan:* {{.notification.data.planName}}
*Payment ID:* {{.notification.data.paymentId}}`

	case string(NotificationTypePaymentFailed):
		return `*{{.notification.title}}* ❌

{{.notification.message}}

*Reason:* {{.notification.data.reason}}
*Payment ID:* {{.notification.data.paymentId}}`

	case string(NotificationTypeCriticalError):
		return `*{{.notification.title}}* 🚨

{{.notification.message}}

*Error Type:* {{.notification.data.errorType}}
*Timestamp:* {{.notification.data.timestamp}}`

	default:
		return `*{{.notification.title}}*

{{.notification.message}}`
	}
}
