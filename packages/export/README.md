# export

CSV and Excel (`.xlsx`) import/export without third-party dependencies.

```go
import (
    "github.com/zatrano/framework/packages/export"
    "github.com/zatrano/framework/packages/export/csv"
    "github.com/zatrano/framework/packages/export/xlsx"
)

// Import (auto-detect by extension)
rows, err := export.ToMaps("people.xlsx", file)

// CSV
text, err := csv.FromMaps(rowsAny)
rows, err = csv.ToMaps(text)
resp := csv.Response("people.csv", rowsAny) // UTF-8 BOM download

// Excel
raw, err := xlsx.FromMaps(rowsAny)
rows, err = xlsx.ToMaps(raw)
resp = xlsx.Response("people.xlsx", rowsAny)
```

Options: `csv.Options{Comma: ';', UseBOM: true}`.
