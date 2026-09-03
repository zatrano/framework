package bootstrap

// EnabledAddons lists first-party addon packages registered by App().
//
// Quick start:
//
//	zatrano package:preset api          // lean API set (+ stubs when available)
//	zatrano package:preset web          // lean web set
//	zatrano package:preset web --merge  // union with current list
//	zatrano package:list
//	zatrano package:enable mongo
//	zatrano package:install billing
//
// Entrypoint: bootstrap.App() reads this list.
// Alternatives: bootstrap.App(bootstrap.Minimal()), App(WithPresetAPI()), App(WithPresetWeb()), App(Kernel()), App(WithAddons(...)).
// Keep this list explicit for production: only enable what the project needs.
var EnabledAddons = []string{}
