package agent

import (
	"context"
	"log"
	"sync"

	"eino-ops-server/agent/tools"
	"eino-ops-server/models"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// knowledgeStore is the package-level knowledge store, set by InitGraph.
var knowledgeStore *gormStore

// agentRunnable is the compiled graph, ready to invoke/stream.
var agentRunnable compose.Runnable[*AgentState, *AgentState]
var initOnce sync.Once

// InitGraph builds the 8-node StateGraph, compiles it, and stores the runnable.
// Must be called once before handling requests.
func InitGraph() {
	initOnce.Do(func() {
		// Initialize store with GORM DB
		knowledgeStore = &gormStore{db: models.DB}

		// Wire toolsSaveToKnowledge
		toolsSaveToKnowledge = func(ctx context.Context, s interface{}, symptoms, diagnosis, rootCause, solution, hosts, tags string) (string, error) {
			return tools.SaveToKnowledge(ctx, knowledgeStore, symptoms, diagnosis, rootCause, solution, hosts, tags)
		}

		// Build tool list
		AllTools = []tool.BaseTool{
			&sshExecTool{resolver: knowledgeStore},
			&sshBatchTool{resolver: knowledgeStore},
			&ipmiPowerTool{resolver: knowledgeStore},
			&ipmiBootdevTool{resolver: knowledgeStore},
			&ipmiResetPwdTool{resolver: knowledgeStore},
			&ipmiSensorTool{resolver: knowledgeStore},
			&ipmiSELTool{resolver: knowledgeStore},
			&checkMonitorTool{store: knowledgeStore},
			&queryDeployTool{store: knowledgeStore},
			&queryLogsTool{resolver: knowledgeStore, store: knowledgeStore},
			&queryKnowledgeTool{store: knowledgeStore},
			&saveKnowledgeTool{store: knowledgeStore},
		}

		g := compose.NewGraph[*AgentState, *AgentState]()

		// Register nodes
		_ = g.AddLambdaNode("understand", compose.InvokableLambda(func(ctx context.Context, s *AgentState) (*AgentState, error) {
			return understandNode(ctx, s)
		}))
		_ = g.AddLambdaNode("plan", compose.InvokableLambda(func(ctx context.Context, s *AgentState) (*AgentState, error) {
			return planNode(ctx, s)
		}))
		_ = g.AddLambdaNode("agent", compose.InvokableLambda(func(ctx context.Context, s *AgentState) (*AgentState, error) {
			return agentNode(ctx, s)
		}))
		_ = g.AddLambdaNode("tool_call", compose.InvokableLambda(func(ctx context.Context, s *AgentState) (*AgentState, error) {
			return toolCallNode(ctx, s)
		}))
		_ = g.AddLambdaNode("observe", compose.InvokableLambda(func(ctx context.Context, s *AgentState) (*AgentState, error) {
			return observeNode(ctx, s)
		}))
		_ = g.AddLambdaNode("human_approve", compose.InvokableLambda(func(ctx context.Context, s *AgentState) (*AgentState, error) {
			return humanApproveNode(ctx, s)
		}))
		_ = g.AddLambdaNode("summarize", compose.InvokableLambda(func(ctx context.Context, s *AgentState) (*AgentState, error) {
			return summarizeNode(ctx, s)
		}))
		_ = g.AddLambdaNode("save_knowledge", compose.InvokableLambda(func(ctx context.Context, s *AgentState) (*AgentState, error) {
			return saveKnowledgeNode(ctx, s)
		}))
		_ = g.AddLambdaNode("backup", compose.InvokableLambda(func(ctx context.Context, s *AgentState) (*AgentState, error) {
			return backupNode(ctx, s)
		}))
		_ = g.AddLambdaNode("audit", compose.InvokableLambda(func(ctx context.Context, s *AgentState) (*AgentState, error) {
			return auditNode(ctx, s)
		}))

		// Entry
		_ = g.AddEdge(compose.START, "understand")

		// Sequential edges
		_ = g.AddEdge("understand", "plan")
		_ = g.AddEdge("plan", "agent")

		// Conditional branch from agent: tool_call / human_approve / summarize
		_ = g.AddBranch("agent", compose.NewGraphBranch(
			func(ctx context.Context, s *AgentState) (string, error) {
				return shouldContinue(s), nil
			},
			map[string]bool{
				"tool_call":     true,
				"human_approve": true,
				"summarize":     true,
			},
		))

		// tool_call → observe → agent (loop)
		_ = g.AddEdge("tool_call", "observe")
		_ = g.AddEdge("observe", "agent")

		// human_approve → backup (approved) or summarize (rejected)
		_ = g.AddBranch("human_approve", compose.NewGraphBranch(
			func(ctx context.Context, s *AgentState) (string, error) {
				if s.Approved {
					return "backup", nil
				}
				return "summarize", nil
			},
			map[string]bool{
				"backup":    true,
				"summarize": true,
			},
		))

		// backup → agent (re-enter loop to execute approved tools)
		_ = g.AddEdge("backup", "agent")

		// summarize → save_knowledge → audit → END
		_ = g.AddEdge("summarize", "save_knowledge")
		_ = g.AddEdge("save_knowledge", "audit")
		_ = g.AddEdge("audit", compose.END)

		var err error
		agentRunnable, err = g.Compile(context.Background())
		if err != nil {
			log.Fatalf("[agent] graph compile failed: %v", err)
		}
		log.Println("[agent] graph compiled successfully")
	})
}

// Invoke runs the graph synchronously (non-streaming).
func Invoke(ctx context.Context, state *AgentState) (*AgentState, error) {
	if agentRunnable == nil {
		InitGraph()
	}
	return agentRunnable.Invoke(ctx, state)
}

// Stream runs the graph and returns a stream reader for SSE events.
func Stream(ctx context.Context, state *AgentState) (*schema.StreamReader[*AgentState], error) {
	if agentRunnable == nil {
		InitGraph()
	}
	return agentRunnable.Stream(ctx, state)
}
