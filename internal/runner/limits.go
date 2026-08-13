package runner

import (
	"errors"
	"fmt"
)

type Enforcement struct{ Memory, Processes, Output, Concurrency, ScopedNetwork bool }

func ValidateEnforcement(spec SandboxSpec, command Command, enforcement Enforcement) error {
	if spec.MemoryBytes <= 0 || !enforcement.Memory {
		return errors.New("sandbox memory limit is not enforceable")
	}
	if spec.ProcessLimit <= 0 || !enforcement.Processes {
		return errors.New("sandbox process limit is not enforceable")
	}
	if command.OutputLimit <= 0 || !enforcement.Output {
		return errors.New("command output limit is not enforceable")
	}
	if !enforcement.Concurrency {
		return errors.New("action concurrency limit is not enforceable")
	}
	if spec.NetworkProfile == "" || spec.NetworkProfile == "unrestricted" || !enforcement.ScopedNetwork {
		return fmt.Errorf("scoped network path %q is not enforceable", spec.NetworkProfile)
	}
	return nil
}
