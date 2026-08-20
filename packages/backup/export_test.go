package backup

// Test helpers (white-box) — not part of the public API.

func RedactSecretsForTest(msg string, secrets ...string) string {
	return redactSecrets(msg, secrets...)
}

func MongoConfigYAMLForTest(m *Manager) (string, []string) {
	return m.mongoConfigYAML()
}
