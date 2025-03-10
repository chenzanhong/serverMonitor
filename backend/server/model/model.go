package model

import (
	"cmd/server/config"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/lib/pq"
)

// 连接数据库并创建表
func InitDB() (*sql.DB, error) { //
	// connStr := "host=192.168.31.251 port=5432 user=postgres password=cCyjKKMyweCer8f3 dbname=monitor sslmode=disable"
	config, _ := config.LoadConfig()
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.DB.Host,
		config.DB.Port,
		config.DB.User,
		config.DB.Password,
		config.DB.Name,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	return db, nil
}

type RequestData struct {
	CPUInfo  []CPUInfo     `json:"cpu_info"`
	HostInfo HostInfo      `json:"host_info"`
	MemInfo  MemoryInfo    `json:"mem_info"`
	ProInfo  []ProcessInfo `json:"pro_info"`
	NetInfo  NetworkInfo   `json:"net_info"`
}

type Claims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

type HostInfo struct {
	ID         int       `json:"id"` // 添加 ID 字段
	Hostname   string    `json:"hostname"`
	OS         string    `json:"os"`
	Platform   string    `json:"platform"`
	KernelArch string    `json:"kernel_arch"`
	CreatedAt  time.Time `json:"host_info_created_at"` // 添加 CreatedAt 字段
	Token      string    `json:"token"`
}

type CPUInfo struct {
	ID        int       `json:"id"` // 添加 ID 字段
	ModelName string    `json:"model_name"`
	CoresNum  int       `json:"cores_num"`
	Percent   float64   `json:"percent"`
	CreatedAt time.Time `json:"cpu_info_created_at"` // 添加 CreatedAt 字段
}

type ProcessInfo struct {
	ID         int       `json:"id"` // 添加 ID 字段
	PID        int       `json:"pid"`
	CPUPercent float64   `json:"cpu_percent"`
	MemPercent float64   `json:"mem_percent"`
	Cmdline    string    `json:"cmdline"`
	CreatedAt  time.Time `json:"pro_info_created_at"` // 添加 CreatedAt 字段
}

type MemoryInfo struct {
	ID          int       `json:"id"` // 添加 ID 字段
	Total       string    `json:"total"`
	Available   string    `json:"available"`
	Used        string    `json:"used"`
	Free        string    `json:"free"`
	UserPercent float64   `json:"user_percent"`
	CreatedAt   time.Time `json:"mem_info_created_at"` // 添加 CreatedAt 字段
}

// 定义网络信息结构体
type NetworkInfo struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	BytesRecv uint64    `json:"bytes_recv"` // 接收字节数
	BytesSent uint64    `json:"bytes_sent"` // 发送字节数
	CreatedAt time.Time `json:"net_info_created_at"`
}

type CPUData struct {
	Time string  `json:"time"`
	Data CPUInfo `json:"data"`
}

type MemoryData struct {
	Time string     `json:"time"`
	Data MemoryInfo `json:"data"`
}

type ProcessData struct {
	Time string      `json:"time"`
	Data ProcessInfo `json:"data"`
}

type NetworkData struct {
	Time string      `json:"time"`
	Data NetworkInfo `json:"data"`
}

func InsertHostInfo(db *sql.DB, hostInfo HostInfo, username string) error {
	var hostInfoID int
	var hostname string
	var exists bool

	// 检查主机记录是否存在
	querySQL := `
    SELECT id, hostname, EXISTS (SELECT 1 FROM host_info WHERE hostname = $1 AND os = $2 AND platform = $3 AND kernel_arch = $4)
    FROM host_info WHERE hostname = $1 AND os = $2 AND platform = $3 AND kernel_arch = $4`

	err := db.QueryRow(querySQL, hostInfo.Hostname, hostInfo.OS, hostInfo.Platform, hostInfo.KernelArch).Scan(&hostInfoID, &hostname, &exists)
	if err == sql.ErrNoRows {
		fmt.Println("No matching host info found.")
		exists = false
	} else if err != nil {
		fmt.Printf("Failed to query host info: %v\n", err)
		return err
	}

	if exists {
		// 更新已存在的主机记录
		updateSQL := `
        UPDATE host_info
        SET host_info_created_at = CURRENT_TIMESTAMP
        WHERE id = $1`
		_, err = db.Exec(updateSQL, hostInfoID)
		if err != nil {
			fmt.Printf("Failed to update host_info_created_at: %v\n", err)
			return err
		}
		fmt.Printf("Updated existing host_info with ID: %d\n", hostInfoID)
	} else {
		// 插入新的主机记录
		insertSQL := `
        INSERT INTO host_info (hostname, os, platform, kernel_arch, host_info_created_at, user_name)
        VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP, $5)
        RETURNING id, hostname`
		err = db.QueryRow(insertSQL, hostInfo.Hostname, hostInfo.OS, hostInfo.Platform, hostInfo.KernelArch, username).Scan(&hostInfoID, &hostname)
		if err != nil {
			fmt.Printf("Failed to insert host_info: %v\n", err)
			return err
		}
		fmt.Printf("Inserted new host_info with ID and Name: %d and %v\n", hostInfoID, hostname)
	}

	return nil
}

func InsertSystemInfo(db *sql.DB, hostInfoID int, cpuInfo CPUInfo, memoryInfo MemoryInfo, processInfo ProcessInfo, networkInfo NetworkInfo) error {
	// 检查是否已经存在对应的 system_info 记录
	var existingID int
	var cpuInfoJSON, memoryInfoJSON, processInfoJSON, networkInfoJSON []byte

	// 查询是否存在
	querySQL := `
	SELECT id, cpu_info, memory_info, process_info, network_info
	FROM system_info
	WHERE host_info_id = $1
	ORDER BY created_at DESC LIMIT 1`

	err := db.QueryRow(querySQL, hostInfoID).Scan(&existingID, &cpuInfoJSON, &memoryInfoJSON, &processInfoJSON, &networkInfoJSON)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to query system_info: %v", err)
	}

	// 获取当前时间并格式化
	currentTime := time.Now().UTC().Format(time.RFC3339)

	// 创建新的数据实例
	cpuData := CPUData{
		Time: currentTime,
		Data: cpuInfo,
	}
	memoryData := MemoryData{
		Time: currentTime,
		Data: memoryInfo,
	}
	processData := ProcessData{
		Time: currentTime,
		Data: processInfo,
	}
	networkData := NetworkData{
		Time: currentTime,
		Data: networkInfo,
	}

	// 处理 CPU 信息
	var cpuInfoArray []CPUData
	if existingID > 0 {
		// 如果已存在记录，反序列化现有的 cpu_info JSON
		if err := json.Unmarshal(cpuInfoJSON, &cpuInfoArray); err != nil {
			return fmt.Errorf("failed to unmarshal existing cpu_info: %v", err)
		}
	}
	// 将新的 CPUData 添加到数组中
	cpuInfoArray = append(cpuInfoArray, cpuData)
	cpuInfoData, err := json.Marshal(cpuInfoArray)
	if err != nil {
		return fmt.Errorf("failed to marshal updated cpu_info: %v", err)
	}

	// 处理 Memory 信息
	var memoryInfoArray []MemoryData
	if existingID > 0 {
		if err := json.Unmarshal(memoryInfoJSON, &memoryInfoArray); err != nil {
			return fmt.Errorf("failed to unmarshal existing memory_info: %v", err)
		}
	}
	memoryInfoArray = append(memoryInfoArray, memoryData)
	memoryInfoData, err := json.Marshal(memoryInfoArray)
	if err != nil {
		return fmt.Errorf("failed to marshal updated memory_info: %v", err)
	}

	// 处理 Process 信息
	var processInfoArray []ProcessData
	if existingID > 0 {
		if err := json.Unmarshal(processInfoJSON, &processInfoArray); err != nil {
			return fmt.Errorf("failed to unmarshal existing process_info: %v", err)
		}
	}
	processInfoArray = append(processInfoArray, processData)
	processInfoData, err := json.Marshal(processInfoArray)
	if err != nil {
		return fmt.Errorf("failed to marshal updated process_info: %v", err)
	}

	// 处理 Network 信息
	var networkInfoArray []NetworkData
	if existingID > 0 {
		if err := json.Unmarshal(networkInfoJSON, &networkInfoArray); err != nil {
			return fmt.Errorf("failed to unmarshal existing network_info: %v", err)
		}
	}
	networkInfoArray = append(networkInfoArray, networkData)
	networkInfoData, err := json.Marshal(networkInfoArray)
	if err != nil {
		return fmt.Errorf("failed to marshal updated network_info: %v", err)
	}

	if existingID > 0 {
		// 更新现有记录
		_, err = db.Exec(`
		UPDATE system_info
		SET cpu_info = $1,
		    memory_info = $2,
		    process_info = $3,
		    network_info = $4,
		    created_at = CURRENT_TIMESTAMP
		WHERE id = $5`,
			cpuInfoData, memoryInfoData, processInfoData, networkInfoData, existingID)
		if err != nil {
			return fmt.Errorf("failed to update system_info: %v", err)
		}
		fmt.Println("Updated existing system_info successfully")
	} else {
		// 插入新的记录
		insertSQL := `
		INSERT INTO system_info (host_info_id, cpu_info, memory_info, process_info, network_info, created_at)
		VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP)`

		_, err := db.Exec(insertSQL, hostInfoID, cpuInfoData, memoryInfoData, processInfoData, networkInfoData)
		if err != nil {
			return fmt.Errorf("failed to insert system_info: %v", err)
		}
		fmt.Println("Inserted new system_info successfully")
	}

	return nil
}

func InsertHostandToken(db *sql.DB, UserName string, Token string) error {

	// 插入新的记录
	fmt.Println("Inserting new host")
	insertSQL := `
	INSERT INTO hostandtoken (host_name, token)
	VALUES ($1, $2) RETURNING token`
	var token string
	err := db.QueryRow(insertSQL, UserName, Token).Scan(&token)
	if err != nil {
		log.Fatalf("Failed to query host info: %v\n", err)
		return err
	}
	log.Println("Insert successfully")

	return nil
}
func ReadMemoryInfo(db *sql.DB, hostname string, from, to string, result map[string]interface{}) error {
	// 查询 JSON 数据
	rows, err := db.Query(`SELECT id, memory_info FROM system_info WHERE hostname = $1`, hostname)
	if err != nil {
		return fmt.Errorf("查询内存信息时发生错误: %v", err)
	}
	defer rows.Close()

	var memoryData []map[string]interface{}

	// 遍历查询结果
	for rows.Next() {
		var id int
		var memInfoJSON []byte

		// 读取查询结果
		err := rows.Scan(&id, &memInfoJSON)
		if err != nil {
			return fmt.Errorf("扫描内存信息记录时发生错误: %v", err)
		}

		// 解析 JSON 数据（假设 mem_info 是一个 JSON 数组）
		var memInfos []map[string]interface{}
		if err := json.Unmarshal(memInfoJSON, &memInfos); err != nil {
			return fmt.Errorf("解析 JSON 数据时发生错误: %v", err)
		}

		// 遍历 JSON 数组中的每个时间点数据
		for _, memInfo := range memInfos {
			// 获取 updated_at 字段
			updatedAtStr, ok := memInfo["updated_at"].(string)
			if !ok {
				continue // 如果 updated_at 字段不存在或类型错误，跳过该记录
			}

			// 将 updated_at 字符串转换为 time.Time
			updatedAt, err := time.Parse(time.RFC3339, updatedAtStr)
			if err != nil {
				return fmt.Errorf("解析 updated_at 字段时发生错误: %v", err)
			}
			fromtime, err := time.Parse(time.RFC3339, from)
			if err != nil {
				return fmt.Errorf("解析 from 字段时发生错误: %v", err)
			}
			totime, err := time.Parse(time.RFC3339, to)
			if err != nil {
				return fmt.Errorf("解析 to 字段时发生错误: %v", err)
			}
			// 判断记录是否在指定时间段内
			if (updatedAt.Equal(fromtime) || updatedAt.After(fromtime)) && updatedAt.Before(totime) {
				memoryData = append(memoryData, memInfo)
			}
		}
	}

	if err = rows.Err(); err != nil {
		return fmt.Errorf("处理内存信息记录时发生错误: %v", err)
	}

	// 将过滤后的数据插入 result
	result["memory"] = memoryData

	return nil
}
func ReadCPUInfo(db *sql.DB, hostname string, from, to string, result map[string]interface{}) error {
	// 查询 JSON 数据
	rows, err := db.Query(`SELECT id, cpu_info FROM system_info WHERE hostname = $1`, hostname)
	if err != nil {
		return fmt.Errorf("查询内存信息时发生错误: %v", err)
	}
	defer rows.Close()

	var cpuData []map[string]interface{}

	// 遍历查询结果
	for rows.Next() {
		var id int
		var cpuJSON []byte

		// 读取查询结果
		err := rows.Scan(&id, &cpuJSON)
		if err != nil {
			return fmt.Errorf("扫描内存信息记录时发生错误: %v", err)
		}

		// 解析 JSON 数据（假设 mem_info 是一个 JSON 数组）
		var cpuInfos []map[string]interface{}
		if err := json.Unmarshal(cpuJSON, &cpuInfos); err != nil {
			return fmt.Errorf("解析 JSON 数据时发生错误: %v", err)
		}

		// 遍历 JSON 数组中的每个时间点数据
		for _, memInfo := range cpuInfos {
			// 获取 updated_at 字段
			updatedAtStr, ok := memInfo["updated_at"].(string)
			if !ok {
				continue // 如果 updated_at 字段不存在或类型错误，跳过该记录
			}

			// 将 updated_at 字符串转换为 time.Time
			updatedAt, err := time.Parse(time.RFC3339, updatedAtStr)
			if err != nil {
				return fmt.Errorf("解析 updated_at 字段时发生错误: %v", err)
			}
			fromtime, err := time.Parse(time.RFC3339, from)
			if err != nil {
				return fmt.Errorf("解析 from 字段时发生错误: %v", err)
			}
			totime, err := time.Parse(time.RFC3339, to)
			if err != nil {
				return fmt.Errorf("解析 to 字段时发生错误: %v", err)
			}
			// 判断记录是否在指定时间段内
			if (updatedAt.Equal(fromtime) || updatedAt.After(fromtime)) && updatedAt.Before(totime) {
				cpuData = append(cpuData, memInfo)
			}
		}
	}

	if err = rows.Err(); err != nil {
		return fmt.Errorf("处理内存信息记录时发生错误: %v", err)
	}

	// 将过滤后的数据插入 result
	result["cpu"] = cpuData

	return nil
}
func ReadNetInfo(db *sql.DB, hostname string, from, to string, result map[string]interface{}) error {
	// 查询 JSON 数据
	rows, err := db.Query(`SELECT id, network_info FROM system_info WHERE hostname = $1`, hostname)
	if err != nil {
		return fmt.Errorf("查询内存信息时发生错误: %v", err)
	}
	defer rows.Close()

	var netData []map[string]interface{}

	// 遍历查询结果
	for rows.Next() {
		var id int
		var netJSON []byte

		// 读取查询结果
		err := rows.Scan(&id, &netJSON)
		if err != nil {
			return fmt.Errorf("扫描内存信息记录时发生错误: %v", err)
		}

		// 解析 JSON 数据（假设 mem_info 是一个 JSON 数组）
		var netInfos []map[string]interface{}
		if err := json.Unmarshal(netJSON, &netInfos); err != nil {
			return fmt.Errorf("解析 JSON 数据时发生错误: %v", err)
		}

		// 遍历 JSON 数组中的每个时间点数据
		for _, netInfo := range netInfos {
			// 获取 updated_at 字段
			updatedAtStr, ok := netInfo["updated_at"].(string)
			if !ok {
				continue // 如果 updated_at 字段不存在或类型错误，跳过该记录
			}

			// 将 updated_at 字符串转换为 time.Time
			updatedAt, err := time.Parse(time.RFC3339, updatedAtStr)
			if err != nil {
				return fmt.Errorf("解析 updated_at 字段时发生错误: %v", err)
			}
			fromtime, err := time.Parse(time.RFC3339, from)
			if err != nil {
				return fmt.Errorf("解析 from 字段时发生错误: %v", err)
			}
			totime, err := time.Parse(time.RFC3339, to)
			if err != nil {
				return fmt.Errorf("解析 to 字段时发生错误: %v", err)
			}
			// 判断记录是否在指定时间段内
			if (updatedAt.Equal(fromtime) || updatedAt.After(fromtime)) && updatedAt.Before(totime) {
				netData = append(netData, netInfo)
			}
		}
	}

	if err = rows.Err(); err != nil {
		return fmt.Errorf("处理内存信息记录时发生错误: %v", err)
	}

	// 将过滤后的数据插入 result
	result["net"] = netData

	return nil
}
func ReadProcessInfo(db *sql.DB, hostname string, from, to string, result map[string]interface{}) error {
	// 查询 JSON 数据
	rows, err := db.Query(`SELECT id, process_info FROM system_info WHERE hostname = $1`, hostname)
	if err != nil {
		return fmt.Errorf("查询内存信息时发生错误: %v", err)
	}
	defer rows.Close()

	var processData []map[string]interface{}

	// 遍历查询结果
	for rows.Next() {
		var id int
		var processJSON []byte

		// 读取查询结果
		err := rows.Scan(&id, &processJSON)
		if err != nil {
			return fmt.Errorf("扫描内存信息记录时发生错误: %v", err)
		}

		// 解析 JSON 数据（假设 mem_info 是一个 JSON 数组）
		var processInfos []map[string]interface{}
		if err := json.Unmarshal(processJSON, &processInfos); err != nil {
			return fmt.Errorf("解析 JSON 数据时发生错误: %v", err)
		}

		// 遍历 JSON 数组中的每个时间点数据
		for _, processInfo := range processInfos {
			// 获取 updated_at 字段
			updatedAtStr, ok := processInfo["updated_at"].(string)
			if !ok {
				continue // 如果 updated_at 字段不存在或类型错误，跳过该记录
			}

			// 将 updated_at 字符串转换为 time.Time
			updatedAt, err := time.Parse(time.RFC3339, updatedAtStr)
			if err != nil {
				return fmt.Errorf("解析 updated_at 字段时发生错误: %v", err)
			}
			fromtime, err := time.Parse(time.RFC3339, from)
			if err != nil {
				return fmt.Errorf("解析 from 字段时发生错误: %v", err)
			}
			totime, err := time.Parse(time.RFC3339, to)
			if err != nil {
				return fmt.Errorf("解析 to 字段时发生错误: %v", err)
			}
			// 判断记录是否在指定时间段内
			if (updatedAt.Equal(fromtime) || updatedAt.After(fromtime)) && updatedAt.Before(totime) {
				processData = append(processData, processInfo)
			}
		}
	}

	if err = rows.Err(); err != nil {
		return fmt.Errorf("处理内存信息记录时发生错误: %v", err)
	}

	// 将过滤后的数据插入 result
	result["process"] = processData

	return nil
}

func ReadDB(db *sql.DB, queryType, from, to string, hostname string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// 查询主机信息
	if queryType == "host" || queryType == "all" {
		row := db.QueryRow("SELECT id, hostname, os, platform, kernel_arch, host_info_created_at FROM host_info WHERE hostname = $1", hostname)
		var id int
		var os, platform, kernelArch string
		var createdAt time.Time
		err := row.Scan(&id, &hostname, &os, &platform, &kernelArch, &createdAt)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, fmt.Errorf("未找到指定的主机记录")
			}
			return nil, fmt.Errorf("查询主机信息时发生错误: %v", err)
		}
		result["host"] = map[string]interface{}{
			"id":                   id,
			"hostname":             hostname,
			"os":                   os,
			"platform":             platform,
			"kernel_arch":          kernelArch,
			"host_info_created_at": createdAt,
		}
	}

	// 查询内存信息
	if queryType == "memory" || queryType == "all" {
		err := ReadMemoryInfo(db, hostname, from, to, result)
		if err != nil {
			return nil, err
		}
	}
	// 查询网卡信息
	if queryType == "net" || queryType == "all" {
		err := ReadNetInfo(db, hostname, from, to, result)
		if err != nil {
			return nil, err
		}
	}
	// 查询 CPU 信息
	if queryType == "cpu" || queryType == "all" {
		err := ReadCPUInfo(db, hostname, from, to, result)
		if err != nil {
			return nil, err
		}
	}

	// 查询进程信息
	if queryType == "process" || queryType == "all" {
		err := ReadProcessInfo(db, hostname, from, to, result)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func UpdateDB(db *sql.DB, host_id int, new_cpu_info []map[string]string, new_memory_info map[string]string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	// 更新CPU信息
	for _, cpu_info := range new_cpu_info {
		_, err = tx.Exec(
			"UPDATE cpu_info SET model_name = $1, cores_num = $2, percent = $3, updated_at = $4 WHERE host_id = $5",
			cpu_info["ModelName"], cpu_info["CoresNum"], cpu_info["Percent"], time.Now(), host_id,
		)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	// 更新内存信息
	_, err = tx.Exec(
		"UPDATE memory_info SET total = $1, available = $2, used = $3, free = $4, user_percent = $5, updated_at = $6 WHERE host_id = $7",
		new_memory_info["Total"], new_memory_info["Available"], new_memory_info["Used"], new_memory_info["Free"], new_memory_info["UserPercent"], time.Now(), host_id,
	)
	if err != nil {
		tx.Rollback()
		return err
	}

	// 提交事务
	err = tx.Commit()
	if err != nil {
		return err
	}

	fmt.Printf("Updated CPU and Memory info for host_id: %d\n", host_id)
	return nil
}

func DeleteDB(db *sql.DB, host_id int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	// 删除CPU信息
	_, err = tx.Exec("DELETE FROM cpu_info WHERE host_id = $1", host_id)
	if err != nil {
		tx.Rollback()
		return err
	}

	// 删除内存信息
	_, err = tx.Exec("DELETE FROM memory_info WHERE host_id = $1", host_id)
	if err != nil {
		tx.Rollback()
		return err
	}

	// 删除进程信息
	_, err = tx.Exec("DELETE FROM process_info WHERE host_id = $1", host_id)
	if err != nil {
		tx.Rollback()
		return err
	}

	// 删除主机信息
	_, err = tx.Exec("DELETE FROM host_info WHERE host_id = $1", host_id)
	if err != nil {
		tx.Rollback()
		return err
	}

	// 提交事务
	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}
