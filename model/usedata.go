package model

import (
	"fmt"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// QuotaData 柱状图数据
type QuotaData struct {
	Id        int    `json:"id"`
	UserID    int    `json:"user_id" gorm:"index"`
	Username  string `json:"username" gorm:"index:idx_qdt_model_user_name,priority:2;size:64;default:''"`
	ModelName string `json:"model_name" gorm:"index:idx_qdt_model_user_name,priority:1;size:64;default:''"`
	CreatedAt int64  `json:"created_at" gorm:"bigint;index:idx_qdt_created_at,priority:2"`
	UseGroup  string `json:"use_group" gorm:"index;size:64;default:''"`
	TokenID   int    `json:"token_id" gorm:"index;default:0"`
	ChannelID int    `json:"channel_id" gorm:"index;default:0"`
	NodeName  string `json:"node_name" gorm:"index;size:64;default:''"`
	TokenUsed int    `json:"token_used" gorm:"default:0"`
	Count     int    `json:"count" gorm:"default:0"`
	Quota     int    `json:"quota" gorm:"default:0"`
}

type QuotaDataLogParams struct {
	UserID    int
	Username  string
	ModelName string
	Quota     int
	CreatedAt int64
	TokenUsed int
	UseGroup  string
	TokenID   int
	ChannelID int
	NodeName  string
}

func UpdateQuotaData() {
	for {
		if common.DataExportEnabled {
			common.SysLog("正在更新数据看板数据...")
			SaveQuotaDataCache()
		}
		time.Sleep(time.Duration(common.DataExportInterval) * time.Minute)
	}
}

var CacheQuotaData = make(map[string]*QuotaData)
var CacheQuotaDataLock = sync.Mutex{}

func logQuotaDataCache(quotaData *QuotaData) {
	key := fmt.Sprintf("%d\x00%s\x00%s\x00%d\x00%s\x00%d\x00%d\x00%s",
		quotaData.UserID,
		quotaData.Username,
		quotaData.ModelName,
		quotaData.CreatedAt,
		quotaData.UseGroup,
		quotaData.TokenID,
		quotaData.ChannelID,
		quotaData.NodeName,
	)
	count := quotaData.Count
	quota := quotaData.Quota
	tokenUsed := quotaData.TokenUsed
	cachedQuotaData, ok := CacheQuotaData[key]
	if ok {
		cachedQuotaData.Count += count
		cachedQuotaData.Quota += quota
		cachedQuotaData.TokenUsed += tokenUsed
		quotaData = cachedQuotaData
	}
	CacheQuotaData[key] = quotaData
}

func LogQuotaData(params QuotaDataLogParams) {
	// 只精确到小时
	createdAt := params.CreatedAt - (params.CreatedAt % 3600)
	quotaData := &QuotaData{
		UserID:    params.UserID,
		Username:  params.Username,
		ModelName: params.ModelName,
		CreatedAt: createdAt,
		UseGroup:  params.UseGroup,
		TokenID:   params.TokenID,
		ChannelID: params.ChannelID,
		NodeName:  params.NodeName,
		Count:     1,
		Quota:     params.Quota,
		TokenUsed: params.TokenUsed,
	}

	CacheQuotaDataLock.Lock()
	defer CacheQuotaDataLock.Unlock()
	logQuotaDataCache(quotaData)
}

func SaveQuotaDataCache() {
	CacheQuotaDataLock.Lock()
	defer CacheQuotaDataLock.Unlock()
	size := len(CacheQuotaData)
	// 如果缓存中有数据，就保存到数据库中
	// 1. 先查询数据库中是否有数据
	// 2. 如果有数据，就更新数据
	// 3. 如果没有数据，就插入数据
	for _, quotaData := range CacheQuotaData {
		quotaDataDB := &QuotaData{}
		DB.Table("quota_data").
			Where("user_id = ? and username = ? and model_name = ? and created_at = ? and use_group = ? and token_id = ? and channel_id = ? and node_name = ?",
				quotaData.UserID, quotaData.Username, quotaData.ModelName, quotaData.CreatedAt, quotaData.UseGroup, quotaData.TokenID, quotaData.ChannelID, quotaData.NodeName).
			First(quotaDataDB)
		if quotaDataDB.Id > 0 {
			//quotaDataDB.Count += quotaData.Count
			//quotaDataDB.Quota += quotaData.Quota
			//DB.Table("quota_data").Save(quotaDataDB)
			increaseQuotaData(quotaData)
		} else {
			DB.Table("quota_data").Create(quotaData)
		}
	}
	CacheQuotaData = make(map[string]*QuotaData)
	common.SysLog(fmt.Sprintf("保存数据看板数据成功，共保存%d条数据", size))
}

func increaseQuotaData(quotaData *QuotaData) {
	err := DB.Table("quota_data").
		Where("user_id = ? and username = ? and model_name = ? and created_at = ? and use_group = ? and token_id = ? and channel_id = ? and node_name = ?",
			quotaData.UserID, quotaData.Username, quotaData.ModelName, quotaData.CreatedAt, quotaData.UseGroup, quotaData.TokenID, quotaData.ChannelID, quotaData.NodeName).
		Updates(map[string]interface{}{
			"count":      gorm.Expr("count + ?", quotaData.Count),
			"quota":      gorm.Expr("quota + ?", quotaData.Quota),
			"token_used": gorm.Expr("token_used + ?", quotaData.TokenUsed),
		}).Error
	if err != nil {
		common.SysLog(fmt.Sprintf("increaseQuotaData error: %s", err))
	}
}

func dashboardHistoricalLogCutoff() (int64, error) {
	result, found, err := loadAtiusDollarCostReconciliationMarker(DB)
	if err != nil {
		return 0, err
	}
	if !found || result.HistoricalLogCutoff <= 0 {
		return 0, nil
	}
	if DB != LOG_DB {
		return 0, fmt.Errorf("historical billing dashboard requires the reconciled primary log database")
	}
	return result.HistoricalLogCutoff, nil
}

func dashboardQuotaRanges(startTime int64, endTime int64) (historicalEnd int64, aggregateStart int64, err error) {
	cutoff, err := dashboardHistoricalLogCutoff()
	if err != nil {
		return 0, 0, err
	}
	if cutoff == 0 {
		return startTime - 1, startTime, nil
	}
	historicalEnd = endTime
	if historicalEnd >= cutoff {
		historicalEnd = cutoff - 1
	}
	aggregateStart = startTime
	if aggregateStart < cutoff {
		aggregateStart = cutoff
	}
	return historicalEnd, aggregateStart, nil
}

func logHourlyCreatedAtExpr() string {
	switch {
	case common.UsingLogDatabase(common.DatabaseTypeMySQL):
		return "FLOOR(created_at / 3600) * 3600"
	case common.UsingLogDatabase(common.DatabaseTypeClickHouse):
		return "intDiv(created_at, 3600) * 3600"
	default:
		return "(created_at / 3600) * 3600"
	}
}

func historicalQuotaLogQuery(startTime int64, endTime int64) *gorm.DB {
	return LOG_DB.Table("logs").
		Where("type = ?", LogTypeConsume).
		Where("created_at >= ? and created_at <= ?", startTime, endTime)
}

func historicalQuotaLogSelect(dimensions string) string {
	return fmt.Sprintf(
		"%s, %s as created_at, count(*) as count, sum(quota) as quota, sum(prompt_tokens + completion_tokens) as token_used",
		dimensions,
		logHourlyCreatedAtExpr(),
	)
}

func historicalQuotaLogGroup(dimensions string) string {
	return fmt.Sprintf("%s, %s", dimensions, logHourlyCreatedAtExpr())
}

func GetQuotaDataByUsername(username string, startTime int64, endTime int64) (quotaData []*QuotaData, err error) {
	var quotaDatas []*QuotaData
	historicalEnd, aggregateStart, err := dashboardQuotaRanges(startTime, endTime)
	if err != nil {
		return nil, err
	}
	if startTime <= historicalEnd {
		dimensions := "user_id, username, model_name"
		var historical []*QuotaData
		if err := historicalQuotaLogQuery(startTime, historicalEnd).
			Select(historicalQuotaLogSelect(dimensions)).
			Where("username = ?", username).
			Group(historicalQuotaLogGroup(dimensions)).
			Find(&historical).Error; err != nil {
			return nil, err
		}
		quotaDatas = append(quotaDatas, historical...)
	}
	if aggregateStart <= endTime {
		var aggregates []*QuotaData
		if err := DB.Table("quota_data").
			Select("user_id, username, model_name, created_at, sum(count) as count, sum(quota) as quota, sum(token_used) as token_used").
			Where("username = ? and created_at >= ? and created_at <= ?", username, aggregateStart, endTime).
			Group("user_id, username, model_name, created_at").
			Find(&aggregates).Error; err != nil {
			return nil, err
		}
		quotaDatas = append(quotaDatas, aggregates...)
	}
	return quotaDatas, err
}

func GetQuotaDataByUserId(userId int, startTime int64, endTime int64) (quotaData []*QuotaData, err error) {
	var quotaDatas []*QuotaData
	historicalEnd, aggregateStart, err := dashboardQuotaRanges(startTime, endTime)
	if err != nil {
		return nil, err
	}
	if startTime <= historicalEnd {
		dimensions := "user_id, username, model_name"
		var historical []*QuotaData
		if err := historicalQuotaLogQuery(startTime, historicalEnd).
			Select(historicalQuotaLogSelect(dimensions)).
			Where("user_id = ?", userId).
			Group(historicalQuotaLogGroup(dimensions)).
			Find(&historical).Error; err != nil {
			return nil, err
		}
		quotaDatas = append(quotaDatas, historical...)
	}
	if aggregateStart <= endTime {
		var aggregates []*QuotaData
		if err := DB.Table("quota_data").
			Select("user_id, username, model_name, created_at, sum(count) as count, sum(quota) as quota, sum(token_used) as token_used").
			Where("user_id = ? and created_at >= ? and created_at <= ?", userId, aggregateStart, endTime).
			Group("user_id, username, model_name, created_at").
			Find(&aggregates).Error; err != nil {
			return nil, err
		}
		quotaDatas = append(quotaDatas, aggregates...)
	}
	return quotaDatas, err
}

func GetQuotaDataGroupByUser(startTime int64, endTime int64) (quotaData []*QuotaData, err error) {
	var quotaDatas []*QuotaData
	historicalEnd, aggregateStart, err := dashboardQuotaRanges(startTime, endTime)
	if err != nil {
		return nil, err
	}
	if startTime <= historicalEnd {
		dimensions := "username"
		var historical []*QuotaData
		if err := historicalQuotaLogQuery(startTime, historicalEnd).
			Select(historicalQuotaLogSelect(dimensions)).
			Group(historicalQuotaLogGroup(dimensions)).
			Find(&historical).Error; err != nil {
			return nil, err
		}
		quotaDatas = append(quotaDatas, historical...)
	}
	if aggregateStart <= endTime {
		var aggregates []*QuotaData
		if err := DB.Table("quota_data").
			Select("username, created_at, sum(count) as count, sum(quota) as quota, sum(token_used) as token_used").
			Where("created_at >= ? and created_at <= ?", aggregateStart, endTime).
			Group("username, created_at").
			Find(&aggregates).Error; err != nil {
			return nil, err
		}
		quotaDatas = append(quotaDatas, aggregates...)
	}
	return quotaDatas, err
}

func GetAllQuotaDates(startTime int64, endTime int64, username string) (quotaData []*QuotaData, err error) {
	if username != "" {
		return GetQuotaDataByUsername(username, startTime, endTime)
	}
	var quotaDatas []*QuotaData
	historicalEnd, aggregateStart, err := dashboardQuotaRanges(startTime, endTime)
	if err != nil {
		return nil, err
	}
	if startTime <= historicalEnd {
		dimensions := "model_name"
		var historical []*QuotaData
		if err := historicalQuotaLogQuery(startTime, historicalEnd).
			Select(historicalQuotaLogSelect(dimensions)).
			Group(historicalQuotaLogGroup(dimensions)).
			Find(&historical).Error; err != nil {
			return nil, err
		}
		quotaDatas = append(quotaDatas, historical...)
	}
	if aggregateStart <= endTime {
		var aggregates []*QuotaData
		if err := DB.Table("quota_data").
			Select("model_name, sum(count) as count, sum(quota) as quota, sum(token_used) as token_used, created_at").
			Where("created_at >= ? and created_at <= ?", aggregateStart, endTime).
			Group("model_name, created_at").
			Find(&aggregates).Error; err != nil {
			return nil, err
		}
		quotaDatas = append(quotaDatas, aggregates...)
	}
	return quotaDatas, err
}
