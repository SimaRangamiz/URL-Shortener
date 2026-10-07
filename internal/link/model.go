package link

import "time"

type Link struct {
	OriginalURL string    `json:"url"`
	Code        string    `json:"code"`
	CreatedAt   time.Time `json:"created_at"`
}