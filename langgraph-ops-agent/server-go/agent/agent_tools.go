package agent

import (
	"context"

	tools "langgraph-ops-server/agent/tools"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// ---- SSH tools ----

type sshExecTool struct {
	resolver toolsHostResolver
}
type sshBatchTool struct {
	resolver toolsHostResolver
}

func (t *sshExecTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "ssh_exec",
		Desc: "Execute a command on a single host via SSH. Returns stdout, stderr, and exit status.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"host": {Type: "string", Desc: "Host name or ID", Required: true},
			"cmd":  {Type: "string", Desc: "Command to execute", Required: true},
		}),
	}, nil
}

func (t *sshExecTool) InvokableRun(ctx context.Context, args string, _ ...tool.Option) (string, error) {
	m := mustParse(args)
	return tools.SshExec(ctx, t.resolver, m["host"], m["cmd"])
}

func (t *sshBatchTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "ssh_batch_exec",
		Desc: "Execute a command on multiple hosts in parallel via SSH. Returns aggregated results.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"hosts": {Type: "array", Desc: "List of host names or IDs", Required: true},
			"cmd":   {Type: "string", Desc: "Command to execute", Required: true},
		}),
	}, nil
}

func (t *sshBatchTool) InvokableRun(ctx context.Context, args string, _ ...tool.Option) (string, error) {
	m := mustParse(args)
	hosts := parseStringList(m["hosts"])
	return tools.SshBatchExec(ctx, t.resolver, hosts, m["cmd"])
}

// ---- IPMI tools (5) ----

type ipmiPowerTool struct{ resolver toolsIPMIResolver }
type ipmiBootdevTool struct{ resolver toolsIPMIResolver }
type ipmiResetPwdTool struct{ resolver toolsIPMIResolver }
type ipmiSensorTool struct{ resolver toolsIPMIResolver }
type ipmiSELTool struct{ resolver toolsIPMIResolver }

func (t *ipmiPowerTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "ipmi_power",
		Desc: "Control host power via IPMI. Action: status/on/off/reset/cycle",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"host":   {Type: "string", Desc: "Host name or ID", Required: true},
			"action": {Type: "string", Desc: "Power action (status/on/off/reset/cycle)", Required: true},
		}),
	}, nil
}
func (t *ipmiPowerTool) InvokableRun(ctx context.Context, args string, _ ...tool.Option) (string, error) {
	m := mustParse(args)
	return tools.IpmiPower(ctx, t.resolver, m["host"], m["action"])
}

func (t *ipmiBootdevTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "ipmi_bootdev",
		Desc: "Set boot device via IPMI. Device: pxe/cdrom/bios/disk",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"host":   {Type: "string", Desc: "Host name or ID", Required: true},
			"device": {Type: "string", Desc: "Boot device (pxe/cdrom/bios/disk)", Required: true},
		}),
	}, nil
}
func (t *ipmiBootdevTool) InvokableRun(ctx context.Context, args string, _ ...tool.Option) (string, error) {
	m := mustParse(args)
	return tools.IpmiBootdev(ctx, t.resolver, m["host"], m["device"])
}

func (t *ipmiResetPwdTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "ipmi_reset_password",
		Desc: "Reset BMC user password via IPMI.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"host":         {Type: "string", Desc: "Host name or ID", Required: true},
			"user_id":      {Type: "string", Desc: "BMC user ID (default 2)", Required: false},
			"new_password": {Type: "string", Desc: "New password (auto-generated if empty)", Required: false},
		}),
	}, nil
}
func (t *ipmiResetPwdTool) InvokableRun(ctx context.Context, args string, _ ...tool.Option) (string, error) {
	m := mustParse(args)
	uid := m["user_id"]
	if uid == "" {
		uid = "2"
	}
	return tools.IpmiResetPassword(ctx, t.resolver, m["host"], uid, m["new_password"])
}

func (t *ipmiSensorTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "ipmi_sensor",
		Desc: "Read all IPMI sensor data (temperature, voltage, fan speed, etc.).",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"host": {Type: "string", Desc: "Host name or ID", Required: true},
		}),
	}, nil
}
func (t *ipmiSensorTool) InvokableRun(ctx context.Context, args string, _ ...tool.Option) (string, error) {
	m := mustParse(args)
	return tools.IpmiSensor(ctx, t.resolver, m["host"])
}

func (t *ipmiSELTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "ipmi_sel",
		Desc: "Read IPMI System Event Log (SEL).",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"host": {Type: "string", Desc: "Host name or ID", Required: true},
		}),
	}, nil
}
func (t *ipmiSELTool) InvokableRun(ctx context.Context, args string, _ ...tool.Option) (string, error) {
	m := mustParse(args)
	return tools.IpmiSEL(ctx, t.resolver, m["host"])
}

// ---- Monitor tools ----

type checkMonitorTool struct{ store toolsMonitorStore }
type queryDeployTool struct{ store toolsMonitorStore }
type queryLogsTool struct {
	resolver toolsHostResolver
	store    toolsMonitorStore
}

func (t *checkMonitorTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "check_monitor",
		Desc: "Query real-time monitoring data for a host (CPU/memory/disk/network/processes).",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"host": {Type: "string", Desc: "Host name or ID", Required: true},
		}),
	}, nil
}
func (t *checkMonitorTool) InvokableRun(ctx context.Context, args string, _ ...tool.Option) (string, error) {
	m := mustParse(args)
	return tools.CheckMonitor(ctx, t.store, m["host"])
}

func (t *queryDeployTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "query_deploy_history",
		Desc: "Query recent deployment/change history for a host.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"host":  {Type: "string", Desc: "Host name or ID", Required: true},
			"hours": {Type: "integer", Desc: "Hours to look back (default 24)", Required: false},
		}),
	}, nil
}
func (t *queryDeployTool) InvokableRun(ctx context.Context, args string, _ ...tool.Option) (string, error) {
	m := mustParse(args)
	hours := 24
	if h, ok := parseInt(m["hours"]); ok {
		hours = h
	}
	return tools.QueryDeployHistory(ctx, t.store, m["host"], hours)
}

func (t *queryLogsTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "query_logs",
		Desc: "Query recent logs from a host via SSH (uses journalctl or tail).",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"host":           {Type: "string", Desc: "Host name or ID", Required: true},
			"service":        {Type: "string", Desc: "Service name (optional)", Required: false},
			"lines":          {Type: "integer", Desc: "Number of lines (default 100)", Required: false},
			"filter_keyword": {Type: "string", Desc: "Filter keyword (optional)", Required: false},
		}),
	}, nil
}
func (t *queryLogsTool) InvokableRun(ctx context.Context, args string, _ ...tool.Option) (string, error) {
	m := mustParse(args)
	lines := 100
	if l, ok := parseInt(m["lines"]); ok {
		lines = l
	}
	cmd := tools.QueryLogs(ctx, m["host"], m["service"], lines, m["filter_keyword"])
	return tools.SshExec(ctx, t.resolver, m["host"], cmd)
}

// ---- Knowledge tools ----

type queryKnowledgeTool struct{ store toolsKnowledgeStore }
type saveKnowledgeTool struct{ store toolsKnowledgeStore }

func (t *queryKnowledgeTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "query_knowledge_base",
		Desc: "Search historical fault cases by symptom or host.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"symptom": {Type: "string", Desc: "Symptom description to search", Required: true},
			"host":    {Type: "string", Desc: "Host name (optional)", Required: false},
		}),
	}, nil
}
func (t *queryKnowledgeTool) InvokableRun(ctx context.Context, args string, _ ...tool.Option) (string, error) {
	m := mustParse(args)
	host := m["host"]
	if host == "" {
		host = "_"
	}
	return tools.QueryKnowledgeBase(ctx, t.store, m["symptom"], host)
}

func (t *saveKnowledgeTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "save_to_knowledge",
		Desc: "Save a resolved case to the knowledge base for future reference.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"symptoms":   {Type: "string", Desc: "Brief symptom description", Required: true},
			"diagnosis":  {Type: "string", Desc: "What was found", Required: true},
			"root_cause": {Type: "string", Desc: "The root cause", Required: true},
			"solution":   {Type: "string", Desc: "How it was fixed", Required: true},
			"hosts":      {Type: "string", Desc: "Affected host names", Required: true},
			"tags":       {Type: "string", Desc: "Comma-separated keywords", Required: true},
		}),
	}, nil
}
func (t *saveKnowledgeTool) InvokableRun(ctx context.Context, args string, _ ...tool.Option) (string, error) {
	m := mustParse(args)
	return tools.SaveToKnowledge(ctx, t.store, m["symptoms"], m["diagnosis"], m["root_cause"], m["solution"], m["hosts"], m["tags"])
}

// ---- Interface aliases for this package ----

type (
	toolsHostResolver  = interface{ Resolve(string) (*tools.SSHConfig, error) }
	toolsIPMIResolver  = interface{ ResolveIPMI(string) (*tools.IPMIConfig, error) }
	toolsKnowledgeStore = interface {
		EnsureTable() error
		Query(symptomPattern, tagPattern, hostPattern string, limit int) ([]tools.KnowledgeCase, error)
		Insert(symptoms, diagnosis, rootCause, solution, hosts, tags string) (int64, error)
	}
	toolsMonitorStore = interface {
		LatestMetric(hostIDOrName string) (*tools.MetricRow, error)
		RecentTasks(hostIDOrName string, hours int, limit int) ([]tools.TaskSummary, error)
	}
)
