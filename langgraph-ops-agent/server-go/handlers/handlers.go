package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"langgraph-ops-server/agent"
	"langgraph-ops-server/middleware"
	"langgraph-ops-server/models"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/bcrypt"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var wsLogClients = make(map[*websocket.Conn]uint)
var wsLogMutex sync.RWMutex

var clientConns = make(map[uint]*websocket.Conn)
var clientConnsMutex sync.RWMutex
var clientWriteMutex sync.Mutex

type TaskCreateRequest struct {
	Name      string `json:"name"`
	ExecType  string `json:"exec_type"`
	Command   string `json:"command"`
	HostIDs   []uint `json:"host_ids"`
	ClientIDs []uint `json:"client_ids"`
}

type ClientHeartbeatRequest struct {
	ClientID uint   `json:"client_id"`
	Name     string `json:"name,omitempty"`
	Host     string `json:"host,omitempty"`
	Status   string `json:"status"`
}

type ClientCmdRequest struct {
	ClientID uint   `json:"client_id"`
	Command  string `json:"command"`
	TaskID   uint   `json:"task_id,omitempty"`
}

type SshConnectRequest struct {
	HostID uint `json:"host_id"`
}

type SshExecuteRequest struct {
	HostID  uint   `json:"host_id"`
	Command string `json:"command"`
}

type AgentExecuteRequest struct {
	TaskID   uint                   `json:"task_id"`
	ExecType string                 `json:"exec_type"`
	Command  string                 `json:"command"`
	HostInfo map[string]interface{} `json:"host_info,omitempty"`
}

type AgentExecuteResponse struct {
	TaskID int      `json:"task_id"`
	Status string   `json:"status"`
	Result string   `json:"result"`
	Logs   []string `json:"logs"`
}

type AgentSshExecuteRequest struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
	AuthType   string `json:"auth_type"`
	Credential string `json:"credential"`
	Command    string `json:"command"`
}

type AgentSshExecuteResponse struct {
	Success bool   `json:"success"`
	Output  string `json:"output,omitempty"`
	Message string `json:"message,omitempty"`
}

type ClientWSMessage struct {
	Type     string `json:"type"`
	TaskID   uint   `json:"task_id,omitempty"`
	Command  string `json:"command,omitempty"`
	Result   string `json:"result,omitempty"`
	Error    string `json:"error,omitempty"`
	ClientID uint   `json:"client_id,omitempty"`
}

type loginRateLimiter struct {
	mu       sync.Mutex
	attempts map[string]int
	resetAt  time.Time
}

var loginLimiter = &loginRateLimiter{
	attempts: make(map[string]int),
	resetAt:  time.Now().Add(1 * time.Minute),
}

const maxLoginAttempts = 10

func (l *loginRateLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if time.Now().After(l.resetAt) {
		l.attempts = make(map[string]int)
		l.resetAt = time.Now().Add(1 * time.Minute)
	}
	if l.attempts[ip] >= maxLoginAttempts {
		return false
	}
	l.attempts[ip]++
	return true
}

func Login(c *gin.Context) {
	clientIP := c.ClientIP()
	if !loginLimiter.allow(clientIP) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many login attempts, try again later"})
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate input lengths
	if len(req.Username) > 100 || len(req.Password) > 200 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}

	var user models.User
	if err := models.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// Compare password: support bcrypt hash or plaintext (auto-upgrade legacy)
	stored := user.Password
	authenticated := false
	if len(stored) >= 4 && stored[:4] == "$2a$" {
		authenticated = bcrypt.CompareHashAndPassword([]byte(stored), []byte(req.Password)) == nil
	} else {
		// Legacy plaintext comparison — auto-upgrade to bcrypt
		authenticated = stored == req.Password
		if authenticated {
			hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
			if hash != nil {
				models.DB.Model(&user).Update("password", string(hash))
			}
		}
	}
	if !authenticated {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	token, err := middleware.GenerateToken(user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":    token,
		"username": user.Username,
	})
}

func GetHosts(c *gin.Context) {
	var hosts []models.Host
	models.DB.Find(&hosts)
	c.JSON(http.StatusOK, gin.H{"data": hosts})
}

type AddHostRequest struct {
	Name       string `json:"name"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
	AuthType   string `json:"auth_type"`
	Credential string `json:"credential"`
}

func AddHost(c *gin.Context) {
	var req AddHostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Port == 0 {
		req.Port = 22
	}
	if req.Port < 1 || req.Port > 65535 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "port must be between 1 and 65535"})
		return
	}
	encryptedCred, err := models.EncryptCredential(req.Credential)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encrypt credential"})
		return
	}
	host := models.Host{
		Name:       req.Name,
		Host:       req.Host,
		Port:       req.Port,
		Username:   req.Username,
		AuthType:   req.AuthType,
		Credential: encryptedCred,
		Status:     "offline",
	}
	models.DB.Create(&host)
	c.JSON(http.StatusOK, gin.H{"data": host})
}

func UpdateHost(c *gin.Context) {
	id := c.Param("id")
	var host models.Host
	if err := models.DB.First(&host, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "host not found"})
		return
	}

	var req struct {
		Name       string `json:"name"`
		Host       string `json:"host"`
		Port       int    `json:"port"`
		Username   string `json:"username"`
		AuthType   string `json:"auth_type"`
		Credential string `json:"credential"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	host.Name = req.Name
	host.Host = req.Host
	host.Port = req.Port
	host.Username = req.Username
	host.AuthType = req.AuthType
	if req.Credential != "" {
		enc, err := models.EncryptCredential(req.Credential)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encrypt credential"})
			return
		}
		host.Credential = enc
	}
	models.DB.Save(&host)
	c.JSON(http.StatusOK, gin.H{"data": host})
}

func DeleteHost(c *gin.Context) {
	id := c.Param("id")
	if err := models.DB.Delete(&models.Host{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func GetTasks(c *gin.Context) {
	var tasks []models.Task
	models.DB.Order("id desc").Find(&tasks)
	c.JSON(http.StatusOK, gin.H{"data": tasks})
}

func CreateTask(c *gin.Context) {
	var req TaskCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name == "" || req.Command == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "任务名称和命令不能为空"})
		return
	}

	if req.ExecType != "ssh" && req.ExecType != "client" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "执行方式必须为 ssh 或 client"})
		return
	}

	if req.ExecType == "ssh" && len(req.HostIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择至少一台主机"})
		return
	}

	if req.ExecType == "client" && len(req.ClientIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择至少一个客户端"})
		return
	}

	task := models.Task{
		Name:     req.Name,
		ExecType: req.ExecType,
		Command:  req.Command,
		Status:   "pending",
	}
	models.DB.Create(&task)

	go processTask(task.ID, req)

	c.JSON(http.StatusOK, gin.H{"data": task})
}

func processTask(taskID uint, req TaskCreateRequest) {
	models.DB.Model(&models.Task{}).Where("id = ?", taskID).Update("status", "running")
	BroadcastLogByTask(taskID, "info", fmt.Sprintf("任务 %d 开始执行", taskID))

	result := ""
	status := "success"
	var err error

	if req.ExecType == "ssh" {
		result, err = dispatchSshTask(taskID, req.HostIDs, req.Command)
		if err != nil {
			status = "failed"
			result = err.Error()
		}
	} else {
		result, err = dispatchClientTask(taskID, req.ClientIDs, req.Command)
		if err != nil {
			status = "failed"
			result = err.Error()
		} else {
			status = "running"
		}
	}

	models.DB.Model(&models.Task{}).Where("id = ?", taskID).Updates(map[string]interface{}{
		"status": status,
		"result": result,
		"logs":   result,
	})

	BroadcastLogByTask(taskID, "info", fmt.Sprintf("任务 %d 执行结束，状态：%s", taskID, status))
}

func dispatchSshTask(taskID uint, hostIDs []uint, command string) (string, error) {
	var builder bytes.Buffer
	failed := false

	for _, hostID := range hostIDs {
		var host models.Host
		if err := models.DB.First(&host, hostID).Error; err != nil {
			msg := fmt.Sprintf("主机 %d 未找到", hostID)
			builder.WriteString(msg + "\n")
			BroadcastLogByTask(taskID, "warn", msg)
			failed = true
			continue
		}

		BroadcastLogByTask(taskID, "info", fmt.Sprintf("开始对主机 %s(%s) 执行命令", host.Name, host.Host))

		resultStr, err := agent.ExecuteSSH(host, command)
		if err != nil {
			msg := fmt.Sprintf("主机 %s SSH 执行失败：%s", host.Name, err.Error())
			builder.WriteString(msg + "\n")
			BroadcastLogByTask(taskID, "error", msg)
			failed = true
			continue
		}

		var sshResult struct {
			Stdout  string `json:"stdout"`
			Stderr  string `json:"stderr"`
			Success bool   `json:"success"`
			Error   string `json:"error"`
		}
		if json.Unmarshal([]byte(resultStr), &sshResult) == nil && sshResult.Success {
			out := sshResult.Stdout
			if sshResult.Stderr != "" {
				out += "\n" + sshResult.Stderr
			}
			builder.WriteString(fmt.Sprintf("[%s] %s\n", host.Name, out))
			BroadcastLogByTask(taskID, "success", fmt.Sprintf("主机 %s 执行完成", host.Name))
		} else {
			errMsg := sshResult.Error
			if errMsg == "" {
				errMsg = sshResult.Stderr
			}
			msg := fmt.Sprintf("主机 %s 执行失败：%s", host.Name, errMsg)
			builder.WriteString(msg + "\n")
			BroadcastLogByTask(taskID, "error", msg)
			failed = true
		}
	}

	result := builder.String()
	if failed {
		return result, errors.New("部分主机执行失败，请检查日志")
	}
	if result == "" {
		result = "SSH 执行完成，无输出"
	}
	return result, nil
}

func dispatchClientTask(taskID uint, clientIDs []uint, command string) (string, error) {
	var builder bytes.Buffer
	sentCount := 0

	for _, clientID := range clientIDs {
		var client models.Client
		if err := models.DB.First(&client, clientID).Error; err != nil {
			msg := fmt.Sprintf("客户端 %d 未找到", clientID)
			builder.WriteString(msg + "\n")
			BroadcastLogByTask(taskID, "warn", msg)
			continue
		}

		conn := getClientConn(client.ID)
		if conn == nil {
			msg := fmt.Sprintf("客户端 %s(%d) 未上线", client.Name, client.ID)
			builder.WriteString(msg + "\n")
			BroadcastLogByTask(taskID, "warn", msg)
			continue
		}

		message := ClientWSMessage{
			Type:    "command",
			TaskID:  taskID,
			Command: command,
		}
		if err := sendClientMessage(conn, message); err != nil {
			msg := fmt.Sprintf("发送命令到客户端 %s 失败：%s", client.Name, err.Error())
			builder.WriteString(msg + "\n")
			BroadcastLogByTask(taskID, "error", msg)
			continue
		}

		builder.WriteString(fmt.Sprintf("已发送命令到客户端 %s(%d)\n", client.Name, client.ID))
		BroadcastLogByTask(taskID, "info", fmt.Sprintf("命令已发送到客户端 %s", client.Name))
		sentCount++
	}

	if sentCount == 0 {
		return builder.String(), errors.New("没有可用客户端，请检查客户端连接")
	}

	return builder.String(), nil
}

func getClientConn(clientID uint) *websocket.Conn {
	clientConnsMutex.RLock()
	defer clientConnsMutex.RUnlock()
	return clientConns[clientID]
}

func addClientConn(clientID uint, ws *websocket.Conn) {
	clientConnsMutex.Lock()
	clientConns[clientID] = ws
	clientConnsMutex.Unlock()
}

func removeClientConn(clientID uint) {
	clientConnsMutex.Lock()
	delete(clientConns, clientID)
	clientConnsMutex.Unlock()
}

func sendClientMessage(ws *websocket.Conn, message ClientWSMessage) error {
	clientWriteMutex.Lock()
	defer clientWriteMutex.Unlock()
	return ws.WriteJSON(message)
}

func GetClients(c *gin.Context) {
	var clients []models.Client
	models.DB.Find(&clients)
	c.JSON(http.StatusOK, gin.H{"data": clients})
}

func AddClient(c *gin.Context) {
	var req struct {
		Name string `json:"name"`
		Host string `json:"host"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	client := models.Client{
		Name:      req.Name,
		Host:      req.Host,
		Status:    "offline",
		LastHeart: time.Now(),
	}
	models.DB.Create(&client)
	c.JSON(http.StatusOK, gin.H{"data": client})
}

func UpdateClient(c *gin.Context) {
	id := c.Param("id")
	var client models.Client
	if err := models.DB.First(&client, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "client not found"})
		return
	}

	var req struct {
		Name string `json:"name"`
		Host string `json:"host"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	client.Name = req.Name
	client.Host = req.Host
	models.DB.Save(&client)
	c.JSON(http.StatusOK, gin.H{"data": client})
}

func DeleteClient(c *gin.Context) {
	id := c.Param("id")
	if err := models.DB.Delete(&models.Client{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func ClientHeartbeat(c *gin.Context) {
	var req ClientHeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var client models.Client
	if err := models.DB.First(&client, req.ClientID).Error; err != nil {
		client = models.Client{
			ID:        req.ClientID,
			Name:      req.Name,
			Host:      req.Host,
			Status:    "online",
			LastHeart: time.Now(),
		}
		models.DB.Create(&client)
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
		return
	}

	client.Name = req.Name
	client.Host = req.Host
	client.Status = req.Status
	client.LastHeart = time.Now()
	models.DB.Save(&client)

	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func SendClientCmd(c *gin.Context) {
	var req ClientCmdRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	conn := getClientConn(req.ClientID)
	if conn == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "client not connected"})
		return
	}

	message := ClientWSMessage{
		Type:    "command",
		TaskID:  req.TaskID,
		Command: req.Command,
	}
	if err := sendClientMessage(conn, message); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "command sent", "client_id": req.ClientID})
}

func SshConnect(c *gin.Context) {
	var req SshConnectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数错误"})
		return
	}

	var host models.Host
	if err := models.DB.First(&host, req.HostID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "主机不存在"})
		return
	}

	// Direct SSH execution
	resultStr, err := agent.ExecuteSSH(host, "echo OK")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	var reply struct {
		Success bool   `json:"success"`
		Output  string `json:"stdout"`
		Error   string `json:"error"`
	}
	if json.Unmarshal([]byte(resultStr), &reply) != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "解析响应失败"})
		return
	}

	if !reply.Success {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": reply.Error})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "output": reply.Output, "message": "连接成功"})
}

func SshExecute(c *gin.Context) {
	var req SshExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数错误"})
		return
	}

	var host models.Host
	if err := models.DB.First(&host, req.HostID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "主机不存在"})
		return
	}

	// Direct SSH execution
	resultStr, err := agent.ExecuteSSH(host, req.Command)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	var reply struct {
		Success bool   `json:"success"`
		Output  string `json:"stdout"`
		Error   string `json:"error"`
	}
	if json.Unmarshal([]byte(resultStr), &reply) != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "解析响应失败"})
		return
	}

	if !reply.Success {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": reply.Error})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "output": reply.Output})
}

func WSLog(c *gin.Context) {
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer ws.Close()

	taskID := uint(0)
	if query := c.Query("task_id"); query != "" {
		if id, err := strconv.Atoi(query); err == nil {
			taskID = uint(id)
		}
	}

	wsLogMutex.Lock()
	wsLogClients[ws] = taskID
	wsLogMutex.Unlock()

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			wsLogMutex.RLock()
			_, ok := wsLogClients[ws]
			wsLogMutex.RUnlock()
			if !ok {
				return
			}
			ws.WriteMessage(websocket.PingMessage, nil)
		}
	}()

	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			break
		}
	}

	wsLogMutex.Lock()
	delete(wsLogClients, ws)
	wsLogMutex.Unlock()
}

func BroadcastLogByTask(taskID uint, level string, message string) {
	wsLogMutex.RLock()
	defer wsLogMutex.RUnlock()

	for ws, filterTask := range wsLogClients {
		if filterTask != 0 && taskID != 0 && filterTask != taskID {
			continue
		}
		ws.WriteJSON(map[string]interface{}{
			"time":    time.Now().Format("15:04:05"),
			"level":   level,
			"message": message,
			"task_id": taskID,
		})
	}
}

func ClientWS(c *gin.Context) {
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer ws.Close()

	clientID := uint(0)
	if header := c.Request.Header.Get("X-Client-ID"); header != "" {
		if id, err := strconv.Atoi(header); err == nil {
			clientID = uint(id)
		}
	}
	if clientID == 0 {
		return
	}

	addClientConn(clientID, ws)
	BroadcastLogByTask(0, "info", fmt.Sprintf("客户端 %d 已连接", clientID))

	for {
		var msg ClientWSMessage
		if err := ws.ReadJSON(&msg); err != nil {
			break
		}
		handleClientMessage(clientID, msg)
	}

	removeClientConn(clientID)
	BroadcastLogByTask(0, "warn", fmt.Sprintf("客户端 %d 已断开", clientID))
}

func handleClientMessage(clientID uint, msg ClientWSMessage) {
	switch msg.Type {
	case "log":
		line := msg.Result
		if line == "" {
			line = msg.Error
		}
		BroadcastLogByTask(msg.TaskID, "info", fmt.Sprintf("客户端 %d: %s", clientID, line))
		appendTaskLog(msg.TaskID, fmt.Sprintf("客户端 %d: %s", clientID, line))
		if msg.Error != "" {
			updateTaskStatus(msg.TaskID, "failed")
		} else if msg.TaskID != 0 {
			updateTaskStatus(msg.TaskID, "success")
		}
	case "heartbeat":
		models.DB.Model(&models.Client{}).Where("id = ?", clientID).Updates(map[string]interface{}{
			"status":     "online",
			"last_heart": time.Now(),
		})
	}
}

func updateTaskStatus(taskID uint, status string) {
	if taskID == 0 {
		return
	}
	models.DB.Model(&models.Task{}).Where("id = ?", taskID).Update("status", status)
}

func appendTaskLog(taskID uint, line string) {
	if taskID == 0 {
		return
	}
	var task models.Task
	if err := models.DB.First(&task, taskID).Error; err != nil {
		return
	}
	logs := task.Logs
	if logs != "" {
		logs += "\n"
	}
	logs += line
	models.DB.Model(&models.Task{}).Where("id = ?", taskID).Update("logs", logs)
}
