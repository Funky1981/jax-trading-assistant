# Security and Compliance

Trading systems handle sensitive data (API keys, account balances) and must comply with financial regulations. Ignoring security can expose you to data breaches, fraud or regulatory penalties.

## Why it matters

Production systems face attackers and must adhere to laws like GDPR, MiFID II and FCA rules. Strong security practices protect your users and your business.

## Tasks

1. **Authentication and authorisation**
   - Implement JWT, OAuth 2.0, or another secure mechanism to authenticate API requests.
   - Use role‑based access control (RBAC) to ensure that only authorised users can place trades or change risk settings.

2. **Secret management**
   - Store API keys, database credentials and broker tokens in a secrets manager (e.g. **HashiCorp Vault**, **AWS Secrets Manager**, **GCP Secret Manager**). Never commit secrets to the codebase.
   - Rotate secrets regularly and audit access logs.

3. **Input validation and sanitisation**
   - Validate all external inputs (HTTP params, file uploads) against strict schemas. Reject or escape unexpected fields.
   - Enforce numeric constraints (e.g. no negative risk percentages) and guard against injection attacks.

4. **Transport security**
   - Serve all APIs over HTTPS. Use TLS certificates from a trusted CA and configure secure ciphers.
   - Validate SSL/TLS certificates when calling external services.

5. **Data privacy and GDPR**
   - Minimise the collection of personal data. Only collect what is strictly necessary for trading.
   - Provide mechanisms for users to request data deletion and to opt out of analytics tracking.
   - Document data retention policies and ensure compliance with the UK Data Protection Act and GDPR.

6. **Audit and compliance**
   - Use the audit logging system to track changes to risk settings, trades and approvals.
   - Implement periodic compliance reviews to ensure trading activities adhere to MiFID II and FCA guidelines.
   - Keep track of regulation changes and update policies accordingly.

7. **Penetration testing and code review**
   - Conduct regular security audits and penetration tests to uncover vulnerabilities.
   - Adopt secure coding practices and peer reviews to catch potential issues before they reach production.
