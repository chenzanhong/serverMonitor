package monitor

import (
	"cmd/server/middlewire"
	"cmd/server/model"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// RequestData 用于接收系统监控数据的请求体
// @Description RequestData 包含所有需要收集的系统信息
type RequestData struct {
	CPUInfo  model.CPUInfo     `json:"cpu_info"`  // CPU 信息
	HostInfo model.HostInfo    `json:"host_info"` // 主机信息
	MemInfo  model.MemoryInfo  `json:"mem_info"`  // 内存信息
	ProInfo  model.ProcessInfo `json:"pro_info"`  // 进程信息
	NetInfo  model.NetworkInfo `json:"net_info"`  // 网络信息
}

// AddSystemInfo 接收并处理系统监控数据
//
// @Summary 接收系统监控信息（CPU、内存、主机信息等）
// @Description 该API用于接收客户端发送的系统监控数据，并验证token和JWT后将数据存储到数据库中。
// @Tags Monitor
// @Accept json
// @Produce json
// @Param request body RequestData true "请求体包含系统监控数据"
// @Success 201 {object} map[string]string "成功响应"
// @Failure 400 {object} map[string]string "无效的JSON数据或令牌长度错误"
// @Failure 401 {object} map[string]string "授权头缺失或无效的token格式或无效的JWT token"
// @Failure 500 {object} map[string]string "数据库操作失败"
// @Router /monitor [post]
func ReceiveAndStoreSystemMetrics(c *gin.Context) {
	// 初始化数据库
	db, err := model.InitDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	} else {
		fmt.Println("Init Successfully")
	}
	defer db.Close()

	// 解析请求数据
	var requestData RequestData
	if err := c.ShouldBindJSON(&requestData); err != nil {
		s := fmt.Sprintf("Invalid JSON data: %s", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": s})
		return
	}
	tokenh := requestData.HostInfo.Token
	var tokens string
	querySQL := `
	SELECT token 
	FROM hostandtoken WHERE hostname = $1` //(SELECT 1 FROM hostandtoken WHERE host_name = $1)

	err = db.QueryRow(querySQL, requestData.HostInfo.Hostname).Scan(&tokens)
	if err == sql.ErrNoRows {
		fmt.Println("No matching token.")
	}
	if len(tokenh) != 16 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Wrong token length"})
		return
	} else if tokenh != tokens {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unequal string"})
		return
	}

	// 从 Authorization Header 中提取 JWT token
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is missing"})
		return
	}
	// 提取 token（格式为 "Bearer <token>"）
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token format"})
		return
	}

	// 解析 token 并验证
	claims := &middlewire.Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return middlewire.JwtKey, nil
	})
	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
		return
	}

	// 更新心跳时间和状态为在线
	updateSQL := `
    UPDATE hostandtoken 
    SET last_heartbeat = NOW(), status = 'online' 
    WHERE host_name = $1`
	_, err = db.Exec(updateSQL, requestData.HostInfo.Hostname)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update heartbeat and status"})
		return
	}

	// 从解析的 token 中获取 username存入数据库
	username := claims.Username

	// 将数据插入数据库
	// 插入 host_info 表
	err = model.InsertHostInfo(db, requestData.HostInfo, username)
	if err != nil {
		s := fmt.Sprintf("Failed to insert system info: %s", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": s})
		return
	}

	// 插入 hostandtoken 表
	err = model.InsertHostandToken(db, requestData.HostInfo.Hostname, tokenh)
	if err != nil {
		s := fmt.Sprintf("Failed to insert system info: %s", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": s})
		return
	}

	// 插入 system_info 表
	err = model.InsertSystemInfo(db, requestData.HostInfo.ID, requestData.CPUInfo, requestData.MemInfo, requestData.ProInfo, requestData.NetInfo)
	if err != nil {
		s := fmt.Sprintf("Failed to insert system info: %s", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": s})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"status": "System information inserted successfully"})
}
