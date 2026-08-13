package evidence

import (
	"bytes"
	"regexp"
)

type Redactor struct{ patterns []*regexp.Regexp }

func NewRedactor() Redactor {
	return Redactor{patterns: []*regexp.Regexp{
		regexp.MustCompile(`(?i)(authorization\s*:\s*bearer\s+)[A-Za-z0-9._~+/=-]+`),
		regexp.MustCompile(`(?i)((?:api[ _-]?key|password|passwd|secret|token)\s*[=:]\s*)[^\s,;]+`),
		regexp.MustCompile(`(?i)(cookie\s*:\s*)[^\r\n]+`),
		regexp.MustCompile(`eyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}`),
		regexp.MustCompile(`(?s)-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----.*?-----END (?:RSA |EC |OPENSSH )?PRIVATE KEY-----`),
	}}
}
func (r Redactor) Bytes(input []byte) []byte {
	output := append([]byte(nil), input...)
	for _, pattern := range r.patterns {
		matches := pattern.FindAllSubmatchIndex(output, -1)
		if len(matches) == 0 {
			continue
		}
		var rebuilt bytes.Buffer
		last := 0
		for _, match := range matches {
			rebuilt.Write(output[last:match[0]])
			if len(match) >= 4 && match[2] >= 0 {
				rebuilt.Write(output[match[2]:match[3]])
			}
			rebuilt.WriteString("[REDACTED]")
			last = match[1]
		}
		rebuilt.Write(output[last:])
		output = rebuilt.Bytes()
	}
	return output
}
func (r Redactor) String(input string) string { return string(r.Bytes([]byte(input))) }
