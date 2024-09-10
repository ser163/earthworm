package feishu

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/larksuite/oapi-sdk-go/v3"
	"github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/larksuite/oapi-sdk-go/v3/service/auth/v3"
	larkbitable "github.com/larksuite/oapi-sdk-go/v3/service/bitable/v1"
	_ "log"
	"net/http"
	"ser163.cn/earthworm/config"
	"sync"
	"time"
)

// FeiShuLib 定义FeiShuLib类
type FeiShuLib struct {
	Client   *lark.Client
	Setting  *config.Config
	Database *sql.DB
	mu       sync.Mutex // 用于并发控制
}

// TenantAccessTokenResponse 返回的结构体
type TenantAccessTokenResponse struct {
	Code              int     `json:"code"`
	Msg               string  `json:"msg"`
	TenantAccessToken string  `json:"tenant_access_token"`
	Expire            float64 `json:"expire"` // 表示过期时间（秒）
}

// NewFeiShuLib 创建FeiShuLib实例
func NewFeiShuLib(db *sql.DB) *FeiShuLib {
	conf := config.GetConfig()
	client := lark.NewClient(
		conf.FeiShu.App.Id, conf.FeiShu.App.Secret,
		lark.WithLogLevel(larkcore.LogLevelDebug),
		lark.WithReqTimeout(3*time.Second),
		lark.WithEnableTokenCache(true),
		lark.WithHelpdeskCredential("id", "token"),
		lark.WithHttpClient(http.DefaultClient))
	return &FeiShuLib{
		Client:   client,
		Setting:  conf,
		Database: db,
	}
}

// ensureTableExists 确保 tokens 表存在
func (f *FeiShuLib) ensureTableExists() error {
	query := `
		CREATE TABLE IF NOT EXISTS tokens (
			id INTEGER PRIMARY KEY,
			token TEXT,
			expires_at DATETIME
		)`
	_, err := f.Database.Exec(query)
	return err
}

// isTokenValid 判断 token 是否有效（有效期剩余大于28分钟）
func (f *FeiShuLib) isTokenValid(expiresAt time.Time) bool {
	return time.Now().Before(expiresAt.Add(-28 * time.Minute))
}

// GetTokenFromDB 从数据库中获取 token 和过期时间
func (f *FeiShuLib) GetTokenFromDB() (string, time.Time, error) {
	// 确保表存在
	if err := f.ensureTableExists(); err != nil {
		return "", time.Time{}, err
	}

	var token string
	var expiresAt time.Time
	query := `SELECT token, expires_at FROM tokens WHERE id = 1`
	err := f.Database.QueryRow(query).Scan(&token, &expiresAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return f.FetchAndSaveToken() // 如果没有找到 token，调用 FetchAndSaveToken 获取新的 token
		}
		return "", time.Time{}, err
	}

	if !f.isTokenValid(expiresAt) {
		return f.FetchAndSaveToken() // 如果 token 不再有效，调用 FetchAndSaveToken 获取新的 token
	}

	fmt.Println("Token fetched from cache")
	return token, expiresAt, nil
}

// saveTokenToDB 将新的 token 保存到数据库
func (f *FeiShuLib) saveTokenToDB(token string, expiresIn float64) error {
	// 确保表存在
	if err := f.ensureTableExists(); err != nil {
		return err
	}

	expiresAt := time.Now().Add(time.Duration(expiresIn) * time.Second)
	query := `INSERT INTO tokens (id, token, expires_at) VALUES (1, ?, ?)
			  ON CONFLICT(id) DO UPDATE SET token=excluded.token, expires_at=excluded.expires_at`
	_, err := f.Database.Exec(query, token, expiresAt)
	return err
}

// FetchAndSaveToken 获取新的 tenant_access_token 并保存到数据库
func (f *FeiShuLib) FetchAndSaveToken() (string, time.Time, error) {
	req := larkauth.NewInternalTenantAccessTokenReqBuilder().
		Body(larkauth.NewInternalTenantAccessTokenReqBodyBuilder().
			AppId(f.Setting.FeiShu.App.Id).
			AppSecret(f.Setting.FeiShu.App.Secret).
			Build()).
		Build()

	// 发起请求
	resp, err := f.Client.Auth.TenantAccessToken.Internal(context.Background(), req)

	// 处理错误
	if err != nil {
		fmt.Println(err)
		return "", time.Time{}, fmt.Errorf("failed to get token: %v", resp.Msg)
	}

	// 服务端错误处理
	if !resp.Success() {
		fmt.Println(resp.Code, resp.Msg, resp.RequestId())
		return "", time.Time{}, fmt.Errorf("failed to get token: %v", resp.Msg)
	}

	// 解析响应
	var result TenantAccessTokenResponse
	if err := json.Unmarshal(resp.RawBody, &result); err != nil {
		return "", time.Time{}, fmt.Errorf("failed to get token: %v", resp.Msg)
	}

	// 获取新的 tenant_access_token 和过期时间
	token := result.TenantAccessToken
	expiresIn := result.Expire // API 返回的过期时间（秒）

	//// 保存新的 token 到数据库
	err = f.saveTokenToDB(token, expiresIn)
	if err != nil {
		return "", time.Time{}, err
	}

	return token, time.Now().Add(time.Duration(expiresIn) * time.Second), nil
}

// GetTenantAccessToken 获取 tenant_access_token，优先从数据库中获取
func (f *FeiShuLib) GetTenantAccessToken() (string, error) {
	token, _, err := f.GetTokenFromDB()
	if err != nil {
		return "", err
	}
	return token, nil
}

// 新建文件记录
func (f *FeiShuLib) NewCreateRecord(record map[string]interface{}) (int, error) {
	token, err := f.GetTenantAccessToken()
	if err != nil {
		return 1, err
	}

	//today := utils.GetToday()
	//
	//record := map[string]interface{}{
	//	"问题描述": "今日头条",
	//	"优先级":  "低 - P2",
	//	"进展状态": "待修复",
	//	"反馈时间": today,
	//}

	// 创建请求对象
	req := larkbitable.NewCreateAppTableRecordReqBuilder().
		AppToken(f.Setting.FeiShu.Drive.BaseId).
		TableId(f.Setting.FeiShu.Drive.TableId).
		AppTableRecord(larkbitable.NewAppTableRecordBuilder().
			Fields(record).
			Build()).
		Build()

	resp, err := f.Client.Bitable.V1.AppTableRecord.Create(context.Background(), req, larkcore.WithTenantAccessToken(token))

	// 处理错误
	if err != nil {
		fmt.Println(err)
		return 1, err
	}

	// 服务端错误处理
	if !resp.Success() {
		// fmt.Println(resp.Code, resp.Msg, resp.RequestId())
		return 1, err
	}

	// 业务处理
	fmt.Println(larkcore.Prettify(resp))

	return 0, nil
}

// 批量新建记录
func (f *FeiShuLib) NewBatchCreateRecord(listRecord []*larkbitable.AppTableRecord) (int, error) {
	if listRecord == nil {
		return 0, nil // Fields is nil
	}

	token, err := f.GetTenantAccessToken()
	if err != nil {
		return 1, err
	}

	// 创建请求对象
	req := larkbitable.NewBatchCreateAppTableRecordReqBuilder().
		AppToken(f.Setting.FeiShu.Drive.BaseId).
		TableId(f.Setting.FeiShu.Drive.TableId).
		Body(larkbitable.NewBatchCreateAppTableRecordReqBodyBuilder().
			Records(listRecord).
			Build()).
		Build()

	resp, err := f.Client.Bitable.AppTableRecord.BatchCreate(context.Background(), req, larkcore.WithTenantAccessToken(token))

	// 处理错误
	if err != nil {
		fmt.Println(err)
		return 1, err
	}

	// 服务端错误处理
	if !resp.Success() {
		fmt.Println(resp.Code, resp.Msg, resp.RequestId())
		return 1, err
	}
	// 业务处理
	fmt.Println(larkcore.Prettify(resp))
	return 0, nil
}
