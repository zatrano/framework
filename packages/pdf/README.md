# pdf

Minimal PDF generation (stdlib only). Use `Inline` for browser viewing or `Attachment` / `Response` for download.

```go
import "github.com/zatrano/framework/packages/pdf"

doc := pdf.New("Report", "Line 1", "Line 2")
bytes := doc.Bytes()

// Table-like text
doc = pdf.FromMaps("Users", []map[string]any{
    {"name": "Ada", "role": "admin"},
}, "name", "role")

return pdf.Inline("users.pdf", doc)      // Content-Disposition: inline
return pdf.Attachment("users.pdf", doc)  // download
```

Not an HTML-to-PDF engine: plain text with automatic page breaks. ASCII uses Helvetica; text with non-ASCII runes embeds a system TrueType font (Arial on Windows, DejaVu/Liberation on Linux) as Identity-H CIDFont so Turkish and other glyphs render.
