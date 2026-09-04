package cookie

import (
	"github.com/zatrano/framework/kernel/encryption"
)

// QueueEncrypted queues an encrypted cookie value.
func (j *Jar) QueueEncrypted(enc *encryption.Encrypter, name, value string, minutes int) error {
	if enc == nil {
		j.Queue(name, value, minutes)
		return nil
	}
	payload, err := enc.Encrypt(value)
	if err != nil {
		return err
	}
	j.Queue(name, payload, minutes)
	return nil
}

// DecryptValue decrypts a cookie payload.
func DecryptValue(enc *encryption.Encrypter, raw string, fallback ...string) string {
	if raw == "" {
		if len(fallback) > 0 {
			return fallback[0]
		}
		return ""
	}
	if enc == nil {
		return raw
	}
	plain, err := enc.Decrypt(raw)
	if err != nil {
		if len(fallback) > 0 {
			return fallback[0]
		}
		return ""
	}
	return plain
}
