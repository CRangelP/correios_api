package handlers

import (
	"net/http"

	"github.com/cleberrangel/correios_api/internal/domain/scraper"
	"github.com/gin-gonic/gin"
)

type TrackerHandler struct {
	scraper scraper.Scraper
}

func NewTrackerHandler(s scraper.Scraper) *TrackerHandler {
	return &TrackerHandler{scraper: s}
}

// TrackCPF godoc
// @Summary Track CPF
// @Description Track package by CPF number
// @Tags tracker
// @Accept json
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Param request body scraper.TrackRequest true "CPF to track"
// @Success 200 {object} scraper.TrackResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/tracker/cpf [post]
func (h *TrackerHandler) TrackCPF(c *gin.Context) {
	var req scraper.TrackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, scraper.TrackResponse{
			Success:        false,
			Error:          "Invalid request: " + err.Error(),
			ScrapingMethod: "browser_automation",
		})
		return
	}

	result, err := h.scraper.TrackCPF(req.CPF)
	if err != nil {
		c.JSON(http.StatusInternalServerError, scraper.TrackResponse{
			Success:        false,
			Error:          "Scraping failed: " + err.Error(),
			ScrapingMethod: "browser_automation",
		})
		return
	}

	c.JSON(http.StatusOK, scraper.TrackResponse{
		Success:        true,
		Data:           result,
		ScrapingMethod: "browser_automation",
	})
}
