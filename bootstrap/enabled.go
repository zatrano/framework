package bootstrap

// EnabledAddons is unused by App(); packages activate by blank-import.
// Kept so generated apps that still assign the list continue to compile.
//
// Lifecycle: imported (init registry) → enabled (WithAddons / all imported) →
// registered → booted (Application.Bootstrap).
var EnabledAddons = []string{}
