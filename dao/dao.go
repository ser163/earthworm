package dao

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"ser163.cn/earthworm/config"
	"strconv"
	"time"
)

// 连接Sqlite
func ConnectDatabase(config *config.Config) (*sql.DB, error) {
	db, err := sql.Open(config.Database.Driver, config.Database.Source)
	if err != nil {
		return nil, err
	}
	return db, nil
}

// 连接mysql 数据库
func ConnectMysqlDatabase(config *config.Config) (*sql.DB, error) {
	host := config.Read.Mysql.Host
	userName := config.Read.Mysql.Username
	pass := config.Read.Mysql.Password
	dbName := config.Read.Mysql.Database
	port := config.Read.Mysql.Port

	// 连接 MySQL 数据库
	dsn := userName + ":" + pass + "@tcp(" + host + ":" + strconv.Itoa(port) + ")/" + dbName
	db, err := sql.Open("mysql", dsn)

	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	return db, nil
}

func CreateTable(db *sql.DB, tableName string) error {
	// 创建包含 token 和过期时间的表
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id INTEGER PRIMARY KEY,
			token TEXT,
			expires_at DATETIME
		)`, tableName)
	_, err := db.Exec(query)
	if err != nil {
		return err
	}
	return nil
}

func ShowTables(db *sql.DB) error {
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table'")
	if err != nil {
		return err
	}
	defer rows.Close()

	fmt.Println("Tables in database:")
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return err
		}
		fmt.Println(name)
	}
	return nil
}

func DropTable(db *sql.DB, tableName string) error {
	query := fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)
	_, err := db.Exec(query)
	if err != nil {
		return err
	}
	return nil
}

// InsertOrUpdateToken 插入或更新token
func InsertOrUpdateToken(db *sql.DB, token string, expiresAt time.Time) error {
	// 更新或插入新的 token 记录
	query := `
		INSERT INTO tokens (id, token, expires_at)
		VALUES (1, ?, ?)
		ON CONFLICT(id) DO UPDATE SET token=excluded.token, expires_at=excluded.expires_at`
	_, err := db.Exec(query, token, expiresAt)
	return err
}

// GetToken 获取当前 token 和其过期时间
func GetToken(db *sql.DB) (string, time.Time, error) {
	var token string
	var expiresAt time.Time
	query := `SELECT token, expires_at FROM tokens WHERE id = 1`
	err := db.QueryRow(query).Scan(&token, &expiresAt)
	if err != nil {
		return "", time.Time{}, err
	}
	return token, expiresAt, nil
}
