package scraper

import "time"

type TrackingEvent struct {
	Date        time.Time `json:"date"`
	Location    string    `json:"location"`
	Description string    `json:"description"`
}

type TrackingResult struct {
	CPF       string          `json:"cpf"`
	Status    string          `json:"status"`
	Events    []TrackingEvent `json:"events"`
	ScrapedAt time.Time       `json:"scraped_at"`
}

type TrackRequest struct {
	CPF string `json:"cpf" binding:"required"`
}

type TrackResponse struct {
	Success        bool            `json:"success"`
	Data           *TrackingResult `json:"data,omitempty"`
	Error          string          `json:"error,omitempty"`
	ScrapingMethod string          `json:"scraping_method"`
}
