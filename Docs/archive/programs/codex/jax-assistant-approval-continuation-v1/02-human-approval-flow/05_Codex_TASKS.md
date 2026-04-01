# Codex Tasks

1. Add approval-related migrations
2. Introduce candidate approval service
3. Create execution instruction model
4. Add approval queue APIs
5. Add expiry and snooze rules
6. Add approval UI page and nav wiring
7. Ensure execution engine accepts approved instructions only
8. Add tests:
   - rejected candidates never execute
   - expired candidates cannot be approved
   - risk-blocked candidate cannot bypass approval
   - fills link back to approval chain
