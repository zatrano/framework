package bootstrap

// EnabledAddons is unused by App() and by the CLI. Packages become available
// by import (init registry); enablement is the consumer manifest registered
// via RegisterEnablement, or DefaultMetas() when no manifest is present.
//
// Kept so generated apps that still assign this identifier in comments or
// copy-paste continue to compile. Do not read it for enablement decisions.
var EnabledAddons = []string{}
