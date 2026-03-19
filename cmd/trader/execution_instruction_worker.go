package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	candidatesmod "jax-trading-assistant/internal/modules/candidates"
	"jax-trading-assistant/internal/modules/execution"
)

func startExecutionInstructionWorker(ctx context.Context, pool *pgxpool.Pool, execService *execution.Service) {
	if pool == nil || execService == nil {
		return
	}

	builder := execution.NewInstructionBuilder(pool)
	candidateStore := candidatesmod.NewStore(pool)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			syncSubmittedInstructions(ctx, pool, builder, candidateStore)
			processNextInstruction(ctx, builder, candidateStore, execService)
		}
	}
}

func processNextInstruction(ctx context.Context, builder *execution.InstructionBuilder, candidateStore *candidatesmod.Store, execService *execution.Service) {
	inst, err := builder.NextPending(ctx)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no rows") {
			return
		}
		log.Printf("execution worker: load pending instruction: %v", err)
		return
	}

	trade, err := execService.ExecuteTrade(ctx, inst.SignalID, inst.ApprovedBy)
	if err != nil {
		_ = builder.MarkRejected(ctx, inst.InstructionID, err.Error())
		_ = candidateStore.UpdateStatus(ctx, inst.CandidateID, candidatesmod.StatusRejected, map[string]any{
			"blockReason": err.Error(),
		})
		publishEvent("execution.rejected", map[string]any{
			"candidateId":   inst.CandidateID,
			"instructionId": inst.InstructionID,
			"signalId":      inst.SignalID,
			"error":         err.Error(),
			"approvedBy":    inst.ApprovedBy,
			"approvedAt":    inst.ApprovedAt,
			"processedAt":   time.Now().UTC(),
		})
		return
	}

	if err := builder.MarkSubmitted(ctx, inst.InstructionID, trade.TradeID.String(), trade.OrderID); err != nil {
		log.Printf("execution worker: mark submitted %s: %v", inst.InstructionID, err)
	}

	status := normalizeExecutionStatus(trade.Status)
	switch status {
	case candidatesmod.StatusFilled:
		if err := builder.MarkFilled(ctx, inst.InstructionID, trade.AvgFillPrice, trade.FilledQty); err != nil {
			log.Printf("execution worker: mark filled %s: %v", inst.InstructionID, err)
		}
		_ = candidateStore.UpdateStatus(ctx, inst.CandidateID, candidatesmod.StatusFilled, nil)
		publishEvent("execution.filled", map[string]any{
			"candidateId":   inst.CandidateID,
			"instructionId": inst.InstructionID,
			"signalId":      inst.SignalID,
			"tradeId":       trade.TradeID.String(),
			"orderId":       trade.OrderID,
			"fillPrice":     trade.AvgFillPrice,
			"fillQty":       trade.FilledQty,
			"filledAt":      time.Now().UTC(),
		})
	default:
		_ = candidateStore.UpdateStatus(ctx, inst.CandidateID, candidatesmod.StatusSubmitted, nil)
		publishEvent("execution.submitted", map[string]any{
			"candidateId":   inst.CandidateID,
			"instructionId": inst.InstructionID,
			"signalId":      inst.SignalID,
			"tradeId":       trade.TradeID.String(),
			"orderId":       trade.OrderID,
			"status":        trade.Status,
			"submittedAt":   time.Now().UTC(),
		})
	}
}

func syncSubmittedInstructions(ctx context.Context, pool *pgxpool.Pool, builder *execution.InstructionBuilder, candidateStore *candidatesmod.Store) {
	rows, err := pool.Query(ctx, `
		SELECT ei.id, ei.candidate_id, COALESCE(ei.trade_id,''), COALESCE(t.order_status,''), COALESCE(t.filled_qty,0), COALESCE(t.avg_fill_price,0)
		FROM execution_instructions ei
		LEFT JOIN trades t ON t.id = ei.trade_id
		WHERE ei.status = 'submitted'
	`)
	if err != nil {
		log.Printf("execution worker: sync submitted instructions: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var instructionID uuid.UUID
		var candidateID uuid.UUID
		var tradeID string
		var orderStatus string
		var fillQty int
		var fillPrice float64
		if err := rows.Scan(&instructionID, &candidateID, &tradeID, &orderStatus, &fillQty, &fillPrice); err != nil {
			continue
		}
		switch normalizeExecutionStatus(orderStatus) {
		case candidatesmod.StatusFilled:
			_ = builder.MarkFilled(ctx, instructionID, fillPrice, fillQty)
			_ = candidateStore.UpdateStatus(ctx, candidateID, candidatesmod.StatusFilled, nil)
			publishEvent("execution.filled", map[string]any{
				"candidateId":   candidateID,
				"instructionId": instructionID,
				"tradeId":       tradeID,
				"fillPrice":     fillPrice,
				"fillQty":       fillQty,
				"filledAt":      time.Now().UTC(),
			})
		case candidatesmod.StatusCancelled:
			_ = builder.MarkCancelled(ctx, instructionID, "trade cancelled by broker")
			_ = candidateStore.UpdateStatus(ctx, candidateID, candidatesmod.StatusCancelled, nil)
			publishEvent("execution.cancelled", map[string]any{
				"candidateId":   candidateID,
				"instructionId": instructionID,
				"tradeId":       tradeID,
				"status":        orderStatus,
				"updatedAt":     time.Now().UTC(),
			})
		case candidatesmod.StatusRejected:
			_ = builder.MarkRejected(ctx, instructionID, "trade rejected by broker")
			_ = candidateStore.UpdateStatus(ctx, candidateID, candidatesmod.StatusRejected, map[string]any{
				"blockReason": "trade rejected by broker",
			})
			publishEvent("execution.rejected", map[string]any{
				"candidateId":   candidateID,
				"instructionId": instructionID,
				"tradeId":       tradeID,
				"status":        orderStatus,
				"updatedAt":     time.Now().UTC(),
			})
		}
	}
}

func normalizeExecutionStatus(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "filled":
		return candidatesmod.StatusFilled
	case "cancelled", "canceled":
		return candidatesmod.StatusCancelled
	case "rejected":
		return candidatesmod.StatusRejected
	case "submitted":
		return candidatesmod.StatusSubmitted
	default:
		return strings.ToLower(strings.TrimSpace(raw))
	}
}

func requireInternalPaperExecute(rContext context.Context, pool *pgxpool.Pool, candidateID uuid.UUID) error {
	if pool == nil || candidateID == uuid.Nil {
		return nil
	}
	var status string
	if err := pool.QueryRow(rContext, `SELECT status FROM candidate_trades WHERE id = $1`, candidateID).Scan(&status); err != nil {
		return fmt.Errorf("candidate lookup: %w", err)
	}
	if status != candidatesmod.StatusApproved && status != candidatesmod.StatusSubmitted && status != candidatesmod.StatusFilled {
		return fmt.Errorf("candidate %s is not approved for execution", candidateID)
	}
	return nil
}
