---
name: validate-object-authorization
description: Use when authenticated Web or API object identifiers may lack owner authorization checks.
license: MIT
domains: [web, api]
intents: [authorization, idor]
---
# Validate Object Authorization

Compare the same object operation using an owner identity, a distinct authorized control identity, and an unauthenticated control when permitted. Treat identifier exposure or HTTP status alone as a lead. Verification requires reproducible cross-identity access to the same protected object, prerequisites, observed data or action impact, and a negative control. State-changing checks require approval.
