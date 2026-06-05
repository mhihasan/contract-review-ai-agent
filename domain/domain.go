package domain

import (
	"fmt"
	"time"
)

type ContractStatus string

const (
	StatusUploaded         ContractStatus = "uploaded"
	StatusExtracting       ContractStatus = "extracting"
	StatusExtracted        ContractStatus = "extracted"
	StatusAnalyzingClauses ContractStatus = "analyzing_clauses"
	StatusClausesExtracted ContractStatus = "clauses_extracted"
	StatusAnalyzing        ContractStatus = "analyzing"
	StatusAnalyzed         ContractStatus = "analyzed"
	StatusReviewPending    ContractStatus = "review_pending"
	StatusReviewComplete   ContractStatus = "review_complete"
	StatusSummarizing      ContractStatus = "summarizing"
	StatusDone             ContractStatus = "done"
)

func (s ContractStatus) String() string { return string(s) }

type RiskLevel string

const (
	RiskHigh   RiskLevel = "high"
	RiskMedium RiskLevel = "medium"
	RiskLow    RiskLevel = "low"
)

func (r RiskLevel) String() string { return string(r) }

func ParseRiskLevel(s string) (RiskLevel, error) {
	switch RiskLevel(s) {
	case RiskHigh, RiskMedium, RiskLow:
		return RiskLevel(s), nil
	}
	return "", fmt.Errorf("invalid risk level: %q", s)
}

type Contract struct {
	ID             string
	Filename       string
	RawText        string
	Status         ContractStatus
	RequiresReview bool
	CreatedAt      time.Time
}

type Clause struct {
	ID             string
	ContractID     string
	SequenceNumber int
	Text           string
}

type ClauseAnalysis struct {
	ID                string
	ClauseID          string
	RiskLevel         *RiskLevel
	Explanation       string
	AmbiguousLanguage string
	Recommendations   string
	Status            string
}

type Review struct {
	ID         string
	ClauseID   string
	Decision   string
	Annotation string
}

type Summary struct {
	ID         string
	ContractID string
	Content    string
	CreatedAt  time.Time
}

type LibraryClause struct {
	ID           string
	ClauseType   string
	StandardText string
	Notes        string
}
