package notification

import "testing"

func TestUseImplicitTLS(t *testing.T) {
	tests := []struct {
		name        string
		encryption  string
		port        string
		wantImplicit bool
	}{
		{name: "tls on 587", encryption: "tls", port: "587", wantImplicit: false},
		{name: "starttls on 587", encryption: "starttls", port: "587", wantImplicit: false},
		{name: "ssl on 465", encryption: "ssl", port: "465", wantImplicit: true},
		{name: "empty on 465", encryption: "", port: "465", wantImplicit: true},
		{name: "TLS uppercase on 587", encryption: "TLS", port: "587", wantImplicit: false},
		{name: "empty on 587", encryption: "", port: "587", wantImplicit: false},
		{name: "ssl on 587", encryption: "ssl", port: "587", wantImplicit: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := useImplicitTLS(SMTPConfig{Encryption: tt.encryption, Port: tt.port})
			if got != tt.wantImplicit {
				t.Fatalf("useImplicitTLS(enc=%q port=%q)=%v want %v", tt.encryption, tt.port, got, tt.wantImplicit)
			}
		})
	}
}
