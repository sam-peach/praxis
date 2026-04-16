package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// matchFeedbackRepository records user accept/reject decisions on similarity candidates.
type matchFeedbackRepository interface {
	record(fb *MatchFeedback, orgID string) error
}

// ── memMatchFeedbackRepository ────────────────────────────────────────────────

type memMatchFeedbackRepository struct {
	mu      sync.Mutex
	entries map[string][]*MatchFeedback // keyed by orgID
}

func newMemMatchFeedbackRepository() *memMatchFeedbackRepository {
	return &memMatchFeedbackRepository{entries: make(map[string][]*MatchFeedback)}
}

func (r *memMatchFeedbackRepository) record(fb *MatchFeedback, orgID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if fb.ID == "" {
		fb.ID = newID()
	}
	fb.OrganizationID = orgID
	if fb.CreatedAt.IsZero() {
		fb.CreatedAt = time.Now().UTC()
	}
	r.entries[orgID] = append(r.entries[orgID], fb)
	return nil
}

func (r *memMatchFeedbackRepository) all(orgID string) []*MatchFeedback {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.entries[orgID]
}

// ── pgMatchFeedbackRepository ─────────────────────────────────────────────────

type pgMatchFeedbackRepository struct {
	db *sql.DB
}

func (r *pgMatchFeedbackRepository) record(fb *MatchFeedback, orgID string) error {
	var bdJSON *string
	if fb.ScoreBreakdown != nil {
		b, err := json.Marshal(fb.ScoreBreakdown)
		if err != nil {
			return fmt.Errorf("marshal score_breakdown: %w", err)
		}
		s := string(b)
		bdJSON = &s
	}

	_, err := r.db.Exec(`
		INSERT INTO match_feedback
			(id, organization_id, drawing_id, candidate_id, action, score, score_breakdown)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		newID(), orgID,
		fb.DrawingID, fb.CandidateID, fb.Action, fb.Score, bdJSON,
	)
	if err != nil {
		log.Printf("match_feedback.record error: %v", err)
		return fmt.Errorf("record feedback: %w", err)
	}
	return nil
}
