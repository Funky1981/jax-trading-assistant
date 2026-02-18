// cmd/artifact-approver is the ADR-0012 Phase 5 approval CLI.
// It walks a strategy artifact through the full approval state machine:
//
//	DRAFT → VALIDATED → REVIEWED → APPROVED
//
// Usage:
//
//	artifact-approver -id strat_momentum_2024-01-15T10:00:00Z \
//	                  -approver alice \
//	                  -notes "metrics look good, risk within bounds"
//
// The -db flag defaults to DATABASE_URL env var if not supplied.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"jax-trading-assistant/internal/domain/artifacts"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	artifactID := flag.String("id", "", "artifact_id string to approve (required)")
	approver := flag.String("approver", "", "approver username or service account (required)")
	notes := flag.String("notes", "", "approval notes or justification")
	dbURL := flag.String("db", os.Getenv("DATABASE_URL"), "postgres connection string (defaults to DATABASE_URL)")
	flag.Parse()

	if *artifactID == "" || *approver == "" {
		fmt.Fprintln(os.Stderr, "error: -id and -approver flags are required")
		flag.Usage()
		os.Exit(1)
	}
	if *dbURL == "" {
		fmt.Fprintln(os.Stderr, "error: DATABASE_URL not set and -db not provided")
		os.Exit(1)
	}

	ctx := context.Background()

	pool, err := connectDB(ctx, *dbURL)
	if err != nil {
		fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	store := artifacts.NewStore(pool)

	// Load artifact by the human-readable string ID.
	artifact, err := store.GetByStringArtifactID(ctx, *artifactID)
	if err != nil {
		fatalf("artifact lookup failed: %v", err)
	}

	// Load current approval state.
	approval, err := store.GetApproval(ctx, artifact.ID)
	if err != nil {
		fatalf("approval lookup failed: %v", err)
	}

	// Display a summary so the approver can make an informed decision.
	printArtifactSummary(artifact, approval)

	if approval.State == artifacts.StateApproved {
		fmt.Println("\nThis artifact is already APPROVED. Nothing to do.")
		return
	}
	if approval.State == artifacts.StateRevoked {
		fmt.Println("\nThis artifact has been REVOKED and cannot be approved.")
		os.Exit(1)
	}

	// Prompt for confirmation.
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("\nApprove artifact %s? (yes/no): ", *artifactID)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer != "yes" && answer != "y" {
		fmt.Println("Aborted — no changes made.")
		os.Exit(0)
	}

	// Walk the state machine: DRAFT → VALIDATED → REVIEWED → APPROVED.
	// UpdateApprovalState enforces ValidTransitions so we must step through each state.
	steps := buildTransitionSteps(approval.State, *approver, *notes)
	for _, step := range steps {
		if err := store.UpdateApprovalState(ctx, artifact.ID, step.toState, step.by, step.reason); err != nil {
			fatalf("failed to transition to %s: %v", step.toState, err)
		}
		fmt.Printf("  ✓ Transitioned to %s\n", step.toState)
	}

	fmt.Printf("\nArtifact %s is now APPROVED.\n", *artifactID)
}

// transitionStep is one hop in the approval chain.
type transitionStep struct {
	toState artifacts.ApprovalState
	by      string
	reason  string
}

// buildTransitionSteps returns the remaining transitions needed to reach APPROVED
// from the current state, honouring the state machine's ValidTransitions.
//
// Chain: DRAFT → VALIDATED → REVIEWED → APPROVED
func buildTransitionSteps(current artifacts.ApprovalState, approver, notes string) []transitionStep {
	chain := []struct {
		from artifacts.ApprovalState
		to   artifacts.ApprovalState
	}{
		{artifacts.StateDraft, artifacts.StateValidated},
		{artifacts.StateValidated, artifacts.StateReviewed},
		{artifacts.StateReviewed, artifacts.StateApproved},
	}

	var steps []transitionStep
	past := false
	for _, hop := range chain {
		if hop.from == current {
			past = true
		}
		if !past {
			continue
		}
		reason := fmt.Sprintf("auto-promoted by artifact-approver (%s)", approver)
		if hop.to == artifacts.StateApproved && notes != "" {
			reason = fmt.Sprintf("approved by %s: %s", approver, notes)
		}
		steps = append(steps, transitionStep{toState: hop.to, by: approver, reason: reason})
	}
	return steps
}

// printArtifactSummary renders an at-a-glance table to stdout.
func printArtifactSummary(a *artifacts.Artifact, ap *artifacts.Approval) {
	fmt.Printf("\n=== Artifact Review ===\n")
	fmt.Printf("  Artifact ID : %s\n", a.ArtifactID)
	fmt.Printf("  Strategy    : %s v%s\n", a.Strategy.Name, a.Strategy.Version)
	fmt.Printf("  Created by  : %s\n", a.CreatedBy)
	fmt.Printf("  Created at  : %s\n", a.CreatedAt.Format(time.RFC3339))
	fmt.Printf("  Hash        : %s\n", a.Hash)
	fmt.Printf("  Approval    : %s (changed by %s at %s)\n",
		ap.State, ap.StateChangedBy, ap.StateChangedAt.Format(time.RFC3339))

	if a.DataWindow != nil {
		fmt.Printf("\n  Data Window : %s → %s\n",
			a.DataWindow.From.Format("2006-01-02"),
			a.DataWindow.To.Format("2006-01-02"))
		fmt.Printf("  Symbols     : %s\n", strings.Join(a.DataWindow.Symbols, ", "))
	}

	if a.Validation != nil && len(a.Validation.Metrics) > 0 {
		fmt.Printf("\n  Backtest Metrics:\n")
		keys := []string{
			"total_trades", "win_rate", "total_return_pct",
			"max_drawdown", "sharpe_ratio", "profit_factor",
		}
		for _, k := range keys {
			if v, ok := a.Validation.Metrics[k]; ok {
				fmt.Printf("    %-22s %v\n", k+":", v)
			}
		}
	}
}

// connectDB opens a pgx pool using the given DSN.
func connectDB(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	return pgxpool.New(ctx, dsn)
}

// fatalf prints to stderr and exits 1.
func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}
