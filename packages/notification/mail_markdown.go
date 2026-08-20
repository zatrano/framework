package notification

import "github.com/zatrano/framework/packages/view"

// SetView attaches a view engine for template-based mail bodies (used by channels).
func (m *MailManager) SetView(engine *view.Engine) {
	if m == nil {
		return
	}
	m.view = engine
}
