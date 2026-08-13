---
name: validate-file-boundaries
description: Use when Web file download upload or path inputs may cross intended storage boundaries.
license: MIT
domains: [web, api]
intents: [file-exposure, upload]
---
# Validate File Boundaries

Establish the intended file namespace, identity, media rules, and storage behavior. Use synthetic canary files and non-executable uploads. A filename reflection or extension match is a lead. Verification requires unauthorized read/write reachability, the exact path or object control, observed impact, and a permitted negative control. Do not upload executable payloads by default.
