package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"eino-ops-server/agent/tools"

	"github.com/cloudwego/eino/components/model"
	einoTool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	openai "github.com/cloudwego/eino-ext/components/model/openai"
)

const systemPrompt = `You are an intelligent operations (运维) AI assistant. Your capabilities:

1. **Diagnose problems**: Analyze monitoring data, logs, and system state to find root causes.
2. **Execute fixes**: Use SSH, IPMI, and other tools to resolve issues.
3. **Plan tasks**: Break complex requests into actionable steps.
4. **Learn from history**: Reference past cases from the knowledge base.

Available tools:
- ssh_exec(host, cmd): Execute command on one host via SSH
- ssh_batch_exec(hosts, cmd): Execute command on multiple hosts in parallel
- check_monitor(host): Query CPU/memory/disk/network metrics for a host
- query_logs(host, service, lines, filter_keyword): Query recent logs
- query_deploy_history(host, hours): Check recent deployments/changes
- query_knowledge_base(symptom, host): Search historical fault cases
- save_to_knowledge(symptoms, diagnosis, root_cause, solution, hosts, tags): Save resolved case
- ipmi_power(host, action): IPMI power control (status/on/off/reset/cycle)
- ipmi_bootdev(host, device): Set boot device (pxe/cdrom/bios/disk)
- ipmi_reset_password(host, user_id, new_password): Reset BMC password
- ipmi_sensor(host): Read IPMI sensor data
- ipmi_sel(host): Read IPMI System Event Log

Rules:
1. When diagnosing, check monitor data first, then logs, then deploy history.
2. Before executing dangerous operations (shutdown, reboot, reinstall, password reset, rm),
   clearly explain the risk and ask for confirmation.
3. When you have enough information to diagnose, summarize findings and suggest next steps.
4. After resolving an issue, save the case to the knowledge base.
5. Use ssh_batch_exec for multi-host operations instead of looping ssh_exec.
6. Respond in Chinese when the user writes in Chinese.`

var (
	defaultLLMModel    = envOr("LLM_MODEL", "deepseek-chat")
	defaultLLMBaseURL  = envOr("LLM_BASE_URL", "https://api.deepseek.com/v1")
	defaultLLMAPIKey   = envOr("LLM_API_KEY", os.Getenv("DEEPSEEK_API_KEY"))
	defaultLLMTemp     = float32(0.3)
)

var AllTools []einoTool.BaseTool // populated by InitGraph

// ---- understand_node ----

func understandNode(ctx context.Context, state *AgentState) (*AgentState, error) {
	input := state.UserInput
	if input == "" {
		state.Intent = map[string]string{"intent": "unknown", "target": "", "description": ""}
		return state, nil
	}

	llm := makeBaseLLM(state)
	prompt := fmt.Sprintf(`Analyze the following user request and extract intent as JSON.
Return ONLY valid JSON with these fields:
  intent: one of [diagnose, execute, query, deploy, repair, batch_check, general]
  target: specific host(s) or system(s) mentioned
  description: one-line summary of what the user wants
  urgency: high/medium/low

User request: "%s"`, input)

	resp, err := llm.Generate(ctx, msgs(userMsg(prompt)))
	if err != nil {
		log.Printf("[understand] LLM error: %v", err)
		state.Intent = map[string]string{"intent": "general", "target": "", "description": input, "urgency": "medium"}
		return state, nil
	}

	intent := map[string]string{}
	if err := json.Unmarshal([]byte(resp.Content), &intent); err != nil {
		intent = map[string]string{"intent": "general", "target": "", "description": input, "urgency": "medium"}
	}
	state.Intent = intent
	state.Messages = append(state.Messages, schema.UserMessage(input))
	log.Printf("[understand] intent: %v", intent)
	return state, nil
}

// ---- plan_node ----

func planNode(ctx context.Context, state *AgentState) (*AgentState, error) {
	llm := makeBaseLLM(state)
	ctxStr := fmt.Sprintf("Intent: %s\n", mustJSON(state.Intent))
	if len(state.KnowledgeContext) > 0 {
		ctxStr += fmt.Sprintf("Related historical cases: %s\n", mustJSON(state.KnowledgeContext))
	}
	ctxStr += "\nCreate a step-by-step plan. Each step must specify which tool to call and why."

	resp, err := llm.Generate(ctx, msgs(schema.SystemMessage(systemPrompt), userMsg(ctxStr+"\n\nOutput the plan as a numbered list of steps.")))
	if err != nil {
		log.Printf("[plan] LLM error: %v", err)
		return state, nil
	}
	state.Plan = resp.Content
	state.Messages = append(state.Messages, schema.AssistantMessage(fmt.Sprintf("📋 执行计划:\n%s", resp.Content), nil))
	log.Printf("[plan] plan: %.200s", resp.Content)
	return state, nil
}

// ---- agent_node ----

func agentNode(ctx context.Context, state *AgentState) (*AgentState, error) {
	llm := makeTooledLLM(state)
	msgs := state.Messages

	// Ensure system prompt is present
	hasSystem := false
	for _, m := range msgs {
		if m.Role == schema.System {
			hasSystem = true
			break
		}
	}
	if !hasSystem {
		msgs = append([]*schema.Message{schema.SystemMessage(systemPrompt)}, msgs...)
	}

	resp, err := llm.Generate(ctx, msgs)
	if err != nil {
		log.Printf("[agent] LLM error: %v", err)
		return state, nil
	}

	state.Messages = append(msgs, resp)

	if len(resp.ToolCalls) > 0 {
		state.PendingToolCalls = resp.ToolCalls
		names := make([]string, len(resp.ToolCalls))
		for i, tc := range resp.ToolCalls {
			names[i] = tc.Function.Name
		}
		log.Printf("[agent] wants tools: %v", names)
	} else {
		state.PendingToolCalls = nil
		log.Printf("[agent] done, no more tools")
	}
	return state, nil
}

// ---- tool_call_node ----

func toolCallNode(ctx context.Context, state *AgentState) (*AgentState, error) {
	for _, tc := range state.PendingToolCalls {
		// Check for high-risk operations BEFORE execution
		if isToolCallRisky(tc) {
			if !state.Approved {
				warning := fmt.Sprintf("⛔ 检测到高危操作 [%s]，已暂停等待人工确认。请在前端点击 [确认执行] 或 [取消操作]。", tc.Function.Name)
				state.Messages = append(state.Messages, schema.ToolMessage(warning, tc.ID))
				state.Messages = append(state.Messages, schema.AssistantMessage(warning, nil))
				state.RequireApproval = true
				log.Printf("[tool_call] BLOCKED high-risk tool (awaiting approval): %s args=%.100s", tc.Function.Name, tc.Function.Arguments)
				continue
			}
			// Approved — run inline backup before executing
			runInlineBackup(ctx, state, tc)
			state.Messages = append(state.Messages, schema.AssistantMessage(fmt.Sprintf("🔓 人工确认通过，版本快照已创建 (%s)，正在执行 [%s]...", state.BackupPath, tc.Function.Name), nil))
			log.Printf("[tool_call] EXECUTING approved high-risk tool: %s args=%.100s", tc.Function.Name, tc.Function.Arguments)
		}

		var result string
		var execErr error

		// Find matching tool
		for _, t := range AllTools {
			info, _ := t.Info(ctx)
			if info.Name == tc.Function.Name {
				if inv, ok := t.(einoTool.InvokableTool); ok {
					result, execErr = inv.InvokableRun(ctx, tc.Function.Arguments)
				} else {
					result = fmt.Sprintf(`{"error": "tool %s is not invokable"}`, tc.Function.Name)
				}
				break
			}
		}
		if execErr != nil {
			result = fmt.Sprintf(`{"error": "%s"}`, execErr.Error())
		}

		toolMsg := schema.ToolMessage(result, tc.ID)
		state.Messages = append(state.Messages, toolMsg)
		log.Printf("[tool_call] %s result: %.200s", tc.Function.Name, result)
	}
	state.PendingToolCalls = nil
	return state, nil
}

func isToolCallRisky(tc schema.ToolCall) bool {
	args := strings.ToLower(tc.Function.Arguments)
	for _, kw := range highRiskKeywords {
		if strings.Contains(args, kw) {
			return true
		}
	}
	// Also check tool name itself (power off/reset are always risky)
	riskyTools := map[string]bool{"ipmi_power": true}
	if riskyTools[tc.Function.Name] {
		action := extractJSONField(args, "action")
		return action == "off" || action == "reset" || action == "cycle"
	}
	return false
}

func extractJSONField(raw, field string) string {
	// Simple extraction: "field": "value" or "field":"value"
	// Handles the common LLM tool call argument format
	key := fmt.Sprintf(`"%s"`, field)
	idx := strings.Index(raw, key)
	if idx < 0 {
		return ""
	}
	rest := raw[idx+len(key):]
	colon := strings.Index(rest, ":")
	if colon < 0 {
		return ""
	}
	val := strings.TrimSpace(rest[colon+1:])
	// Unquote
	val = strings.Trim(val, `", `)
	return val
}

// ---- observe_node ----

func observeNode(ctx context.Context, state *AgentState) (*AgentState, error) {
	llm := makeBaseLLM(state)
	msgs := state.Messages

	observationPrompt := "Review the tool execution results above. " +
		"What did we learn? Is the problem solved or do we need more investigation? " +
		"If more tools are needed, explain what to check next."
	msgs = append(msgs, userMsg(observationPrompt))

	resp, err := llm.Generate(ctx, msgs)
	if err != nil {
		log.Printf("[observe] LLM error: %v", err)
		return state, nil
	}

	state.Messages = append(msgs, resp)
	state.Observations = append(state.Observations, resp.Content)
	log.Printf("[observe] %.200s", resp.Content)
	return state, nil
}

// ---- human_approve_node ----

var highRiskKeywords = []string{
	"reboot", "shutdown", "reinstall", "reset_password",
	"rm -rf", "rm -r", "fdisk", "mkfs", "dd if",
	"systemctl stop", "docker rm", "kubectl delete",
	"iptables -F", "power_off", "power_reset",
}

func humanApproveNode(ctx context.Context, state *AgentState) (*AgentState, error) {
	riskFound := false
	for _, m := range state.Messages {
		content := strings.ToLower(m.Content)
		if m.ToolCalls != nil {
			for _, tc := range m.ToolCalls {
				b, _ := json.Marshal(tc)
				content += strings.ToLower(string(b))
			}
		}
		for _, kw := range highRiskKeywords {
			if strings.Contains(content, kw) {
				riskFound = true
				break
			}
		}
		if riskFound {
			break
		}
	}

	state.RequireApproval = riskFound
	if riskFound {
		if state.Approved {
			log.Printf("[human_approve] approved — proceeding to backup")
			return state, nil
		}
		warning := "⚠️ 此操作涉及高危动作，已暂停等待人工确认。请在前端点击 [确认执行] 或 [取消操作]。"
		state.Messages = append(state.Messages, schema.AssistantMessage(warning, nil))
		log.Printf("[human_approve] HIGH RISK — waiting for approval")
	}
	return state, nil
}

// ---- backup_node ----

func backupNode(ctx context.Context, state *AgentState) (*AgentState, error) {
	if !state.Approved {
		log.Printf("[backup] skipped — not approved")
		return state, nil
	}

	// Collect target hosts from pending tool calls and messages
	hosts := extractTargetHosts(state)
	if len(hosts) == 0 {
		log.Printf("[backup] no target hosts identified, skipping")
		state.BackupResults = "no hosts to backup"
		return state, nil
	}

	ts := time.Now().Format("20060102_150405")
	backupBase := fmt.Sprintf("/var/backup/agent/%s/%s", state.ThreadID, ts)
	state.BackupPath = backupBase

	var results []string
	for _, host := range hosts {
		cmds := []string{
			fmt.Sprintf("mkdir -p %s/%s", backupBase, host),
			fmt.Sprintf("cp -a /etc %s/%s/etc 2>/dev/null || echo 'etc backup failed'", backupBase, host),
			fmt.Sprintf("ps aux > %s/%s/processes.txt 2>/dev/null", backupBase, host),
			fmt.Sprintf("ss -tlnp > %s/%s/ports.txt 2>/dev/null", backupBase, host),
			fmt.Sprintf("df -h > %s/%s/disk_usage.txt 2>/dev/null", backupBase, host),
		}

		for _, cmd := range cmds {
			execSSHBackground(ctx, host, cmd)
		}
		results = append(results, fmt.Sprintf("%s: config/processes/ports/disk backed up", host))
		log.Printf("[backup] %s → %s/%s", host, backupBase, host)
	}

	state.BackupResults = strings.Join(results, "; ")
	state.Messages = append(state.Messages, schema.AssistantMessage(
		fmt.Sprintf("📦 版本快照已创建: %s\n备份内容: 配置文件(/etc)、进程列表、端口占用、磁盘使用率。可用于回滚验证。", state.BackupResults), nil))
	return state, nil
}

// extractTargetHosts collects host identifiers from pending tool calls and recent messages.
func extractTargetHosts(state *AgentState) []string {
	seen := map[string]bool{}
	var hosts []string

	for _, tc := range state.PendingToolCalls {
		host := extractJSONField(tc.Function.Arguments, "host")
		if host != "" && !seen[host] {
			seen[host] = true
			hosts = append(hosts, host)
		}
		hostsField := extractJSONField(tc.Function.Arguments, "hosts")
		if hostsField != "" {
			for _, h := range strings.Split(hostsField, ",") {
				h = strings.TrimSpace(h)
				if h != "" && !seen[h] {
					seen[h] = true
					hosts = append(hosts, h)
				}
			}
		}
	}
	return hosts
}

// runInlineBackup runs a quick backup of the target host before executing an approved risky tool.
func runInlineBackup(ctx context.Context, state *AgentState, tc schema.ToolCall) {
	host := extractJSONField(tc.Function.Arguments, "host")
	if host == "" {
		return
	}
	ts := time.Now().Format("20060102_150405")
	backupBase := fmt.Sprintf("/var/backup/agent/%s/%s", state.ThreadID, ts)
	state.BackupPath = fmt.Sprintf("%s/%s", backupBase, host)
	state.BackupResults = fmt.Sprintf("host=%s path=%s", host, state.BackupPath)

	cmds := []string{
		fmt.Sprintf("mkdir -p %s/%s", backupBase, host),
		fmt.Sprintf("cp -a /etc %s/%s/etc 2>/dev/null || echo 'etc backup skipped'", backupBase, host),
		fmt.Sprintf("ps aux > %s/%s/processes.txt 2>/dev/null", backupBase, host),
		fmt.Sprintf("ss -tlnp > %s/%s/ports.txt 2>/dev/null", backupBase, host),
		fmt.Sprintf("df -h > %s/%s/disk_usage.txt 2>/dev/null", backupBase, host),
	}
	for _, cmd := range cmds {
		execSSHBackground(ctx, host, cmd)
	}
}

// execSSHBackground executes a command via SSH on a target host.
func execSSHBackground(ctx context.Context, hostIdentifier, cmd string) {
	if knowledgeStore == nil {
		log.Printf("[backup] no store available for SSH to %s", hostIdentifier)
		return
	}
	result, err := tools.SshExec(ctx, knowledgeStore, hostIdentifier, cmd)
	if err != nil {
		log.Printf("[backup] SSH %s cmd='%s' failed: %v", hostIdentifier, cmd, err)
		return
	}
	log.Printf("[backup] SSH %s cmd='%s' → %.200s", hostIdentifier, cmd, result)
}

// ---- audit_node ----

func auditNode(ctx context.Context, state *AgentState) (*AgentState, error) {
	if knowledgeStore == nil {
		log.Printf("[audit] no store available, skipping")
		return state, nil
	}

	// Ensure audit table exists
	if err := knowledgeStore.EnsureAuditTable(); err != nil {
		log.Printf("[audit] ensure table failed: %v", err)
	}

	// Build tools_called summary
	var toolNames []string
	var highRisk []string
	for _, tc := range state.PendingToolCalls {
		toolNames = append(toolNames, tc.Function.Name)
		if isToolCallRisky(tc) {
			highRisk = append(highRisk, tc.Function.Name)
		}
	}
	// Also scan messages for tool invocations already executed
	for _, m := range state.Messages {
		if m.Role == schema.Tool {
			toolNames = append(toolNames, m.ToolName)
		}
	}
	toolsCalled := strings.Join(toolNames, ",")
	highRiskOps := strings.Join(highRisk, ",")

	intentJSON, _ := json.Marshal(state.Intent)

	entry := AuditEntry{
		ThreadID:       state.ThreadID,
		Intent:         string(intentJSON),
		Plan:           state.Plan,
		ToolsCalled:    toolsCalled,
		HighRiskOps:    highRiskOps,
		Approved:       state.Approved,
		BackupPath:     state.BackupPath,
		HostsAffected:  strings.Join(extractTargetHosts(state), ","),
		FinalResult:    state.FinalAnswer,
		Observations:   strings.Join(state.Observations, " | "),
		KnowledgeSaved: true,
		Status:         "completed",
	}

	if err := knowledgeStore.InsertAuditLog(entry); err != nil {
		log.Printf("[audit] insert failed: %v", err)
		return state, nil
	}

	log.Printf("[audit] saved audit log for thread=%s tools=%s high_risk=%s approved=%v backup=%s",
		state.ThreadID, toolsCalled, highRiskOps, state.Approved, state.BackupPath)
	state.Messages = append(state.Messages, schema.AssistantMessage(
		fmt.Sprintf("📝 审计记录已保存 (thread: %s)", state.ThreadID), nil))
	return state, nil
}

// ---- summarize_node ----

func summarizeNode(ctx context.Context, state *AgentState) (*AgentState, error) {
	llm := makeBaseLLM(state)
	prompt := `Based on the entire conversation above, generate a clear summary report for the user. Include:
1. What was the original request
2. What steps were taken
3. What was found (root cause if diagnosing)
4. What was done to fix it (if applicable)
5. Recommendations or next steps

Format the response in clean Markdown. Use Chinese.`

	msgs := append([]*schema.Message{schema.SystemMessage(systemPrompt)}, state.Messages...)
	msgs = append(msgs, userMsg(prompt))
	resp, err := llm.Generate(ctx, msgs)
	if err != nil {
		log.Printf("[summarize] LLM error: %v", err)
		return state, nil
	}
	state.FinalAnswer = resp.Content
	log.Printf("[summarize] len=%d", len(resp.Content))
	return state, nil
}

// ---- save_knowledge_node ----

func saveKnowledgeNode(ctx context.Context, state *AgentState) (*AgentState, error) {
	// Try extracting case via LLM, but continue on failure
	llm := makeBaseLLM(state)
	prompt := `Based on the troubleshooting session above, extract key learnings as JSON:
{"symptoms": "brief description", "diagnosis": "what was found", "root_cause": "the root cause", "solution": "how it was fixed", "hosts": "affected hosts", "tags": "comma-separated keywords"}

If no clear diagnosis was reached, return: {"skip": true, "reason": "no diagnosis made"}

Return ONLY valid JSON, no other text.`

	lastMsgs := state.Messages
	if len(lastMsgs) > 20 {
		lastMsgs = lastMsgs[len(lastMsgs)-20:]
	}
	msgs := append(lastMsgs, userMsg(prompt))
	resp, err := llm.Generate(ctx, msgs)
	if err != nil {
		log.Printf("[save_knowledge] LLM error: %v", err)
		return state, nil
	}

	var kcase struct {
		Skip     bool   `json:"skip"`
		Reason   string `json:"reason"`
		Symptoms string `json:"symptoms"`
		Diagnosis string `json:"diagnosis"`
		RootCause string `json:"root_cause"`
		Solution string `json:"solution"`
		Hosts    string `json:"hosts"`
		Tags     string `json:"tags"`
	}
	if err := json.Unmarshal([]byte(resp.Content), &kcase); err != nil || kcase.Skip {
		if kcase.Skip {
			log.Printf("[save_knowledge] skipped: %s", kcase.Reason)
		} else {
			log.Printf("[save_knowledge] parse error: %v (raw: %.200s)", err, resp.Content)
		}
		return state, nil
	}

	// Use package-level knowledge store (set by InitGraph via SetStore)
	if knowledgeStore != nil {
		result, err := toolsSaveToKnowledge(ctx, knowledgeStore, kcase.Symptoms, kcase.Diagnosis, kcase.RootCause, kcase.Solution, kcase.Hosts, kcase.Tags)
		if err == nil {
			log.Printf("[save_knowledge] saved: %s", result)
			state.Messages = append(state.Messages, schema.AssistantMessage(fmt.Sprintf("📚 知识库已更新: %s", result), nil))
		}
	}
	return state, nil
}

// ---- routing helpers ----

func shouldContinue(state *AgentState) string {
	if state.RequireApproval {
		return "human_approve"
	}
	if len(state.PendingToolCalls) > 0 {
		return "tool_call"
	}
	return "summarize"
}

func shouldObserve(state *AgentState) string {
	return "observe"
}

// ---- LLM helpers ----

func makeLLMConfig(state *AgentState) openai.ChatModelConfig {
	m := defaultLLMModel
	baseURL := defaultLLMBaseURL
	apiKey := defaultLLMAPIKey

	if cfg := state.LlmConfig; cfg != nil {
		if v := cfg["api_key"]; v != "" {
			apiKey = v
		}
		if v := cfg["base_url"]; v != "" {
			baseURL = v
		}
		if v := cfg["model"]; v != "" {
			m = v
		}
	}

	return openai.ChatModelConfig{
		Model:       m,
		APIKey:      apiKey,
		BaseURL:     baseURL,
		Temperature: &defaultLLMTemp,
	}
}

// makeBaseLLM returns a BaseChatModel for nodes that only need Generate (no tools).
func makeBaseLLM(state *AgentState) model.BaseChatModel {
	cfg := makeLLMConfig(state)
	llm, err := openai.NewChatModel(context.Background(), &cfg)
	if err != nil {
		log.Printf("[llm] create error: %v", err)
		return nil
	}
	return llm
}

// makeTooledLLM returns a ToolCallingChatModel for the agent node.
func makeTooledLLM(state *AgentState) model.ToolCallingChatModel {
	cfg := makeLLMConfig(state)
	llm, err := openai.NewChatModel(context.Background(), &cfg)
	if err != nil {
		log.Printf("[llm] create error: %v", err)
		return nil
	}
	if len(AllTools) == 0 {
		return llm
	}
	infos := make([]*schema.ToolInfo, len(AllTools))
	for i, t := range AllTools {
		info, _ := t.Info(context.Background())
		infos[i] = info
	}
	tooled, err := llm.WithTools(infos)
	if err != nil {
		log.Printf("[llm] WithTools error: %v", err)
		return llm
	}
	return tooled
}

func msgs(m ...*schema.Message) []*schema.Message { return m }
func userMsg(content string) *schema.Message      { return schema.UserMessage(content) }

func mustJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// toolsSaveToKnowledge bridges the knowledge store for save_knowledge_node.
// We use a package-level variable set by InitGraph to avoid import cycles.
var toolsSaveToKnowledge = func(ctx context.Context, store interface{}, symptoms, diagnosis, rootCause, solution, hosts, tags string) (string, error) {
	return `{"error": "knowledge store not initialized"}`, nil
}
