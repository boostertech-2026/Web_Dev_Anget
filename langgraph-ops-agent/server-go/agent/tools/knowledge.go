package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// KnowledgeCase is a resolved ops case stored for future reference.
type KnowledgeCase struct {
	ID         uint      `json:"id"`
	Symptoms   string    `json:"symptoms"`
	Diagnosis  string    `json:"diagnosis"`
	RootCause  string    `json:"root_cause"`
	Solution   string    `json:"solution"`
	Hosts      string    `json:"hosts"`
	Tags       string    `json:"tags"`
	ResolvedAt string    `json:"resolved_at"`
	CreatedAt  time.Time `json:"created_at"`
}

// KnowledgeStore abstracts knowledge persistence so tools don't import GORM directly.
type KnowledgeStore interface {
	Query(symptomPattern, tagPattern, hostPattern string, limit int) ([]KnowledgeCase, error)
	Insert(symptoms, diagnosis, rootCause, solution, hosts, tags string) (int64, error)
	EnsureTable() error
}

func escapeLike(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}

// QueryKnowledgeBase searches historical fault cases by symptom or host.
func QueryKnowledgeBase(ctx context.Context, store KnowledgeStore, symptom, host string) (string, error) {
	if err := store.EnsureTable(); err != nil {
		return "", err
	}
	pattern := "%" + escapeLike(symptom) + "%"
	cases, err := store.Query(pattern, pattern, "%"+escapeLike(host)+"%", 5)
	if err != nil {
		return "", err
	}
	if len(cases) == 0 {
		return `{"found": false, "message": "No similar historical cases found"}`, nil
	}
	b, _ := json.Marshal(map[string]interface{}{
		"found": true,
		"cases": cases,
		"count": len(cases),
	})
	return string(b), nil
}

// SaveToKnowledge stores a resolved case.
func SaveToKnowledge(ctx context.Context, store KnowledgeStore, symptoms, diagnosis, rootCause, solution, hosts, tags string) (string, error) {
	if err := store.EnsureTable(); err != nil {
		return "", err
	}
	id, err := store.Insert(symptoms, diagnosis, rootCause, solution, hosts, tags)
	if err != nil {
		return "", fmt.Errorf("failed to save knowledge: %w", err)
	}
	b, _ := json.Marshal(map[string]interface{}{
		"success": true,
		"case_id": id,
		"message": "Knowledge saved",
	})
	return string(b), nil
}
