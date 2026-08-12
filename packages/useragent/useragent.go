package useragent

import (
	"strings"
)

// Agent is a parsed User-Agent summary.
type Agent struct {
	Raw      string `json:"raw"`
	Browser  string `json:"browser"`
	Platform string `json:"platform"`
	Device   string `json:"device"`
	IsMobile bool   `json:"is_mobile"`
	IsBot    bool   `json:"is_bot"`
}

// Parse extracts coarse browser/platform/device info from a User-Agent string.
func Parse(ua string) Agent {
	raw := strings.TrimSpace(ua)
	lower := strings.ToLower(raw)
	a := Agent{Raw: raw, Browser: "Unknown", Platform: "Unknown", Device: "Desktop"}

	bots := []string{"bot", "spider", "crawler", "slurp", "facebookexternalhit", "preview"}
	for _, b := range bots {
		if strings.Contains(lower, b) {
			a.IsBot = true
			a.Browser = "Bot"
			a.Device = "Bot"
			break
		}
	}

	switch {
	case strings.Contains(lower, "edg/"):
		a.Browser = "Edge"
	case strings.Contains(lower, "chrome/") && !strings.Contains(lower, "edg/"):
		a.Browser = "Chrome"
	case strings.Contains(lower, "firefox/"):
		a.Browser = "Firefox"
	case strings.Contains(lower, "safari/") && !strings.Contains(lower, "chrome/"):
		a.Browser = "Safari"
	case strings.Contains(lower, "msie") || strings.Contains(lower, "trident/"):
		a.Browser = "IE"
	}

	switch {
	case strings.Contains(lower, "android"):
		a.Platform = "Android"
		a.IsMobile = true
		a.Device = "Mobile"
	case strings.Contains(lower, "iphone") || strings.Contains(lower, "ipod"):
		a.Platform = "iOS"
		a.IsMobile = true
		a.Device = "Mobile"
	case strings.Contains(lower, "ipad"):
		a.Platform = "iOS"
		a.IsMobile = true
		a.Device = "Tablet"
	case strings.Contains(lower, "windows"):
		a.Platform = "Windows"
	case strings.Contains(lower, "mac os") || strings.Contains(lower, "macintosh"):
		a.Platform = "macOS"
	case strings.Contains(lower, "linux"):
		a.Platform = "Linux"
	}

	if strings.Contains(lower, "mobile") || strings.Contains(lower, "mobi") {
		a.IsMobile = true
		if a.Device == "Desktop" {
			a.Device = "Mobile"
		}
	}
	if strings.Contains(lower, "tablet") {
		a.Device = "Tablet"
		a.IsMobile = true
	}
	return a
}
