# Third-party dependencies

CyberPilot tracks direct Go dependencies here. Release automation must regenerate or verify this inventory before publishing.

| Dependency | Purpose | License |
|---|---|---|
| `github.com/google/uuid` | UUIDv7 identifiers | BSD-3-Clause |
| `github.com/gofrs/flock` | Cross-platform daemon lock | BSD-3-Clause |
| `github.com/Microsoft/go-winio` | Windows named-pipe transport | MIT |
| `modernc.org/sqlite` | Pure-Go SQLite event store | BSD-3-Clause |
| `gopkg.in/yaml.v3` | Versioned YAML configuration and Skill metadata | Apache-2.0 / MIT |
| `golang.org/x/term` | Hidden terminal credential input | BSD-3-Clause |
| `github.com/charmbracelet/bubbletea` | Terminal application runtime | MIT |
| `github.com/charmbracelet/lipgloss` | Responsive terminal rendering | MIT |

Transitive dependencies and future embedded assets must be included before a tagged release. This file is an inventory, not a replacement for upstream license notices.
