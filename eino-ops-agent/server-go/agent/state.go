package agent

import "github.com/cloudwego/eino/schema"

// AgentState is the shared state passed through every node in the graph.
type AgentState struct {
	Messages         []*schema.Message
	UserInput        string
	Intent           map[string]string
	Plan             string
	PendingToolCalls []schema.ToolCall
	Observations     []string
	FinalAnswer      string
	RequireApproval  bool
	Approved         bool // set by frontend after user confirms high-risk operation
	BackupPath       string
	BackupResults    string
	KnowledgeContext []map[string]string
	LlmConfig        map[string]string
	ThreadID         string
}

// Clone returns a shallow copy (slices are shared, not deep-copied).
func (s *AgentState) Clone() *AgentState {
	c := *s
	return &c
}
