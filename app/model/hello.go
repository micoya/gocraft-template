package model

import "time"

// Hello 打招呼的数据模型
type Hello struct {
	ID        int64     `json:"id"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}
