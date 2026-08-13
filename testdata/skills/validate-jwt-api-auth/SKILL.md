---
name: validate-jwt-api-auth
description: Use when an API accepts bearer JWT tokens and claim or signature trust needs validation.
license: MIT
domains: [api]
intents: [authentication, jwt]
---
# Validate JWT API Authentication

Record the accepted issuer, audience, algorithm, expiry, and identity claims before proposing a minimal controlled variation. A decoded token or rejected request is only a lead. Verification requires a server-accepted unauthorized identity or privilege effect plus an unchanged-token control. Never expose bearer values to model context, logs, or exports.
