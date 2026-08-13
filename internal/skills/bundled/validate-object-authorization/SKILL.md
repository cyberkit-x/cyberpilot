---
name: validate-object-authorization
description: Use when Web or API object identifiers may lack owner authorization checks.
license: MIT
domains: [web, api]
intents: [authorization, idor]
---
# Validate Object Authorization

Compare the same object operation with an owner identity and a distinct control identity. A status code alone is a lead. Verification requires reproducible cross-identity access, observed protected data or state impact, prerequisites, and negative control evidence.
