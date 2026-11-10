package domain

import "time"

// Record 是各业务表共用的主键和时间戳。ID 继续使用已发布的数字 JSON。
type Record struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
