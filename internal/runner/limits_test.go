package runner

import "testing"

func TestResourceAndNetworkEnforcementFailsClosed(t *testing.T) {
	spec := SandboxSpec{MemoryBytes: 512 << 20, ProcessLimit: 128, NetworkProfile: "broker:session"}
	command := Command{OutputLimit: 1 << 20}
	all := Enforcement{Memory: true, Processes: true, Output: true, Concurrency: true, ScopedNetwork: true}
	if err := ValidateEnforcement(spec, command, all); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		modify func(*SandboxSpec, *Command, *Enforcement)
	}{{"memory", func(s *SandboxSpec, c *Command, e *Enforcement) { e.Memory = false }}, {"process", func(s *SandboxSpec, c *Command, e *Enforcement) { s.ProcessLimit = 0 }}, {"output", func(s *SandboxSpec, c *Command, e *Enforcement) { c.OutputLimit = 0 }}, {"concurrency", func(s *SandboxSpec, c *Command, e *Enforcement) { e.Concurrency = false }}, {"network", func(s *SandboxSpec, c *Command, e *Enforcement) { s.NetworkProfile = "unrestricted" }}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, c, e := spec, command, all
			tt.modify(&s, &c, &e)
			if err := ValidateEnforcement(s, c, e); err == nil {
				t.Fatal("unenforceable configuration accepted")
			}
		})
	}
}
