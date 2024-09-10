package read

import (
	"database/sql"
	"errors"
	larkbitable "github.com/larksuite/oapi-sdk-go/v3/service/bitable/v1"
	"log"
	"ser163.cn/earthworm/config"
	"ser163.cn/earthworm/utils"
	"strings"
	"time"
)

// ReadLib 定义ReadLib类
type ReadLib struct {
	Setting  *config.Config
	Database *sql.DB
	SqlLite  *sql.DB
	Begin    int64
	End      int64
}

// ReadLib 创建ReadLib实例
func NewReadLib(mysqldb *sql.DB, sqllite *sql.DB) *ReadLib {
	conf := config.GetConfig()
	return &ReadLib{
		Setting:  conf,
		Database: mysqldb,
		SqlLite:  sqllite,
		Begin:    0,
		End:      0,
	}
}

// 加工字段
func (r *ReadLib) Transfer() ([]*larkbitable.AppTableRecord, error) {
	// 确保表存在
	if err := r.ensureTableExists(); err != nil {
		return nil, err
	}
	// 获取线上最后一条记录
	remoteLastId, err := r.getLastId()
	if err != nil {
		log.Println(err)
		return nil, err
	}
	println("remote server id: ", remoteLastId)
	// 获取本地最后一条记录id
	localLastId, err := r.getLocalLastId()
	if err != nil {
		if err == sql.ErrNoRows {
			localLastId = 0
		} else {
			log.Println(err)
			return nil, err
		}
	}
	println("local id: ", localLastId)
	// 对比本地和远程id
	if localLastId > remoteLastId {
		return nil, errors.New("The local last id must be smaller than the remote service last id")
	}
	// 计算差值,如果太大,则进行报错
	var difference = remoteLastId - localLastId
	println("difference: ", difference)
	if difference > r.Setting.Read.Mode.Rows {
		return nil, errors.New("The difference is too big, please handle it manually")
	}

	ids := utils.GenerateIDList(localLastId, remoteLastId)
	if ids == nil || len(ids) == 0 {
		return nil, nil
	}

	records, err := r.fetchRecords(ids)
	if err != nil {
		return nil, err
	}

	if len(records) > 0 {
		appTableRecords, err := r.feildToFormatArray(records)
		if err != nil {
			return nil, err
		}
		r.Begin = localLastId
		r.End = remoteLastId
		return appTableRecords, nil
	}

	return nil, nil
}

// 将[]map[string]interface{} 转换为 []*larkbitable.AppTableRecord
func (r *ReadLib) feildToFormatArray(orgRecords []map[string]interface{}) ([]*larkbitable.AppTableRecord, error) {
	tableRecords := make([]*larkbitable.AppTableRecord, 0)
	for _, record := range orgRecords {
		args := make(map[string]interface{})
		args["需求描述"] = record["des"]
		args["需求分类"] = "用户需求反馈"
		args["需求状态"] = "待评估"
		args["优先级"] = "低 - P2"
		createTime, err := utils.TimeStrToUnixMilli(record["add_date"].(string))
		if err != nil {
			return nil, err
		}
		args["需求提出日期"] = createTime // 这里把add_date作为转换

		email := strings.TrimSpace(record["email"].(string))
		if email != "" {
			email = " 联系方式: " + email
		}

		desData := record["des"].(string) + email
		args["需求详细描述（可附文档）"] = desData
		// args["父记录"] = ["reculp3iz80VL5"]
		args["父记录"] = []string{}
		args["父记录"] = append(args["父记录"].([]string), "recumeyGcqvGUP")
		record := &larkbitable.AppTableRecord{
			Fields:           args,
			CreatedTime:      utils.GetNowUnixMilli(),
			LastModifiedTime: utils.GetNowUnixMilli(),
		}
		tableRecords = append(tableRecords, record)
	}
	return tableRecords, nil
}

// FetchRecords 根据ID列表从数据库中查询记录
func (f *ReadLib) fetchRecords(ids []int64) ([]map[string]interface{}, error) {
	// 构造 SQL 查询
	query := `SELECT id, des, email, user_id, add_date FROM book_user_feedback WHERE id IN (` + utils.BuildPlaceholders(len(ids)) + `)`

	// 将 ids 转换为 interface{} 切片，以传递给 Query
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	// 执行查询
	rows, err := f.Database.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// 存储结果
	var records []map[string]interface{}

	for rows.Next() {
		var id int64
		var des string
		var email string
		var user_id int64
		var add_date string

		err := rows.Scan(&id, &des, &email, &user_id, &add_date)
		if err != nil {
			return nil, err
		}

		// 保存每条记录
		record := map[string]interface{}{
			"id":       id,
			"des":      des,
			"email":    email,
			"user_id":  user_id,
			"add_date": add_date,
		}

		records = append(records, record)
	}

	// 检查是否有查询错误
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return records, nil
}

// ensureTableExists 确保 records 存在
func (r *ReadLib) ensureTableExists() error {
	query := `
		CREATE TABLE IF NOT EXISTS records (
			id INTEGER PRIMARY KEY,
			feed_id INTEGER,
			flag INTEGER DEFAULT 0,
			created_at DATETIME
		)`
	_, err := r.SqlLite.Exec(query)

	// 为 feed_id 创建索引
	_, err = r.SqlLite.Exec(`CREATE INDEX IF NOT EXISTS idx_feed_id ON records (feed_id)`)
	if err != nil {
		log.Fatal(err)
	}

	// 为 flag 创建索引
	_, err = r.SqlLite.Exec(`CREATE INDEX IF NOT EXISTS idx_flag ON records (flag)`)
	if err != nil {
		log.Fatal(err)
	}

	return err
}

// 获取MySql Read中最后一条id
func (r *ReadLib) getLastId() (int64, error) {
	var id int64
	query := `SELECT id FROM book_user_feedback order by id desc limit 1`
	err := r.Database.QueryRow(query).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *ReadLib) getLocalLastId() (int64, error) {
	var feed_id int64
	query := `SELECT feed_id FROM records where flag = 0 order by id desc limit 1`
	err := r.SqlLite.QueryRow(query).Scan(&feed_id)
	if err != nil {
		return 0, err
	}
	return feed_id, nil
}

// 更新本地结果
func (r *ReadLib) UploadLocalRecord() error {
	if r.Begin == r.End {
		log.Fatal("no record get update")
		return nil
	}

	// 开启事务
	tx, err := r.SqlLite.Begin()
	if err != nil {
		log.Fatal(err)
		return err
	}

	// 更新 开始记录
	begin, err := tx.Prepare("UPDATE records SET flag = ? WHERE feed_id = ?")
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer begin.Close()

	// 执行更新操作
	_, err = begin.Exec(1, r.Begin)
	if err != nil {
		// 如果有错误，回滚事务
		tx.Rollback()
		log.Fatal(err)
		return err
	}

	end, err := tx.Prepare("INSERT INTO records(feed_id, flag, created_at) VALUES(?, ?, ?)")
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer end.Close()

	// 获取当前时间
	currentTime := time.Now()

	// 格式化为 SQLite 支持的格式（ISO 8601 格式）
	formattedDateTime := currentTime.Format("2006-01-02 15:04:05")

	// 执行更新操作
	_, err = end.Exec(r.End, 0, formattedDateTime)
	if err != nil {
		// 如果有错误，回滚事务
		tx.Rollback()
		log.Fatal(err)
		return err
	}

	// 提交事务
	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
		return err
	}
	return nil
}
