---
name: validate-scoped-ssrf
description: Use when a Web or API feature fetches user-controlled URLs within confirmed test scope.
license: MIT
domains: [web, api]
intents: [ssrf, redirect]
---
# Validate Scoped SSRF

Use only operator-controlled local callback endpoints inside confirmed scope. Revalidate DNS answers and every redirect destination. A timing change or generic error is a lead. Verification requires a unique callback or response attributable to the target-side fetch and a non-fetching control. Never probe cloud metadata or private addresses outside explicit scope.
