# 02 Human Approval Flow

Goal: candidate trades become executable only after explicit human approval.

This continues from the current system because `frontend_api.go` already has:
- `POST /api/v1/signals/{id}/approve`
- `POST /api/v1/signals/{id}/reject`

This folder upgrades that from basic signal approval into a structured candidate-trade approval workflow.
