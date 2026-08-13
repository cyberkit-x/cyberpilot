package evidence

import (
	"strings"
	"testing"
)

func TestRedactionAcrossOutwardBoundaries(t *testing.T) {
	redactor := NewRedactor()
	secrets := []string{"Authorization: Bearer abc.def.ghiSECRET", "api_key=sk-live-secret", "password: hunter2", "Cookie: session=sensitive", "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.signaturevalue", "-----BEGIN PRIVATE KEY-----\nprivate-data\n-----END PRIVATE KEY-----"}
	for _, secret := range secrets {
		for _, boundary := range []string{"model context", "log", "terminal", "export"} {
			output := redactor.String("prefix " + secret + " suffix")
			if !strings.Contains(output, "[REDACTED]") || strings.Contains(output, "hunter2") || strings.Contains(output, "private-data") || strings.Contains(output, "sensitive") {
				t.Fatalf("boundary=%s input=%q output=%q", boundary, secret, output)
			}
		}
	}
}
func TestProtectedRawArtifactRemainsByteExact(t *testing.T) {
	raw := []byte("Authorization: Bearer preserve-local-raw")
	stored := append([]byte(nil), raw...)
	_ = NewRedactor().Bytes(raw)
	if string(stored) != string(raw) {
		t.Fatal("protected raw artifact was modified")
	}
}
