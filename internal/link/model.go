package link

import "time"

type Link struct {
	Code        string    `json:"code" gorm:"primaryKey;type:varchar(10)"`
	OriginalURL string    `json:"url" gorm:"index;type:text;not null"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
}