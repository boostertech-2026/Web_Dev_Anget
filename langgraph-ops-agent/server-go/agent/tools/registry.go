package tools

// This file is deliberately empty.
// Tool registration happens in agent/agent_tools.go where we wire everything
// together with concrete GORM-backed store implementations.
//
// The *Store interfaces in each tool file (KnowledgeStore, MonitorStore,
// HostResolver, IPMIResolver) are the extension points — the agent package
// provides GORM-backed implementations and registers the tools.
