package memory

type Kind string

const (
	KindProfile    Kind = "profile"
	KindPreference Kind = "preference"
	KindProject    Kind = "project"
	KindWorkflow   Kind = "workflow"
	KindDecision   Kind = "decision"
	KindConnection Kind = "connection"
	KindNote       Kind = "note"
)

type Confidence string

const (
	ConfidenceConfirmed Confidence = "confirmed"
	ConfidenceInferred  Confidence = "inferred"
	ConfidenceTemporary Confidence = "temporary"
)

type Record struct {
	ID         int64      `json:"id"`
	Scope      string     `json:"scope"`
	Kind       Kind       `json:"kind"`
	Content    string     `json:"content"`
	Confidence Confidence `json:"confidence"`
	CreatedAt  string     `json:"created_at"`
	UpdatedAt  string     `json:"updated_at"`
	Embedding  []float32  `json:"-"`
}
