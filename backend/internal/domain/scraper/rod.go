package scraper

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

type RodScraper struct {
	browser *rod.Browser
}

func NewRodScraper(browserURL string) (*RodScraper, error) {
	var browser *rod.Browser

	if browserURL != "" {
		browser = rod.New().ControlURL(browserURL).MustConnect()
	} else {
		path, found := launcher.LookPath()
		if !found {
			path = "/usr/bin/google-chrome"
		}
		log.Printf("Using browser at: %s", path)
		u := launcher.New().Bin(path).Headless(true).NoSandbox(true).MustLaunch()
		browser = rod.New().ControlURL(u).MustConnect()
	}

	return &RodScraper{browser: browser}, nil
}

func (r *RodScraper) TrackCPF(cpf string) (*TrackingResult, error) {
	log.Printf("Starting tracking for CPF: %s", cpf)

	page := r.browser.MustPage()
	page.MustSetUserAgent(&proto.NetworkSetUserAgentOverride{
		UserAgent: "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	})

	defer func() {
		if err := page.Close(); err != nil {
			log.Printf("Error closing page: %v", err)
		}
	}()

	page.Timeout(30 * time.Second)

	err := page.Navigate("https://www.haga7digital.com.br/?page=rastreio")
	if err != nil {
		return nil, fmt.Errorf("failed to navigate: %w", err)
	}

	log.Println("Page navigated, waiting for load...")
	page.MustWaitLoad()
	time.Sleep(3 * time.Second)

	log.Println("Looking for CPF input...")
	cpfInput, err := page.Element("input")
	if err != nil {
		return nil, fmt.Errorf("failed to find CPF input: %w", err)
	}

	log.Println("Filling CPF...")
	cpfInput.MustInput(cpf)
	time.Sleep(500 * time.Millisecond)

	log.Println("Looking for submit button...")
	submitBtn, err := page.Element("button")
	if err != nil {
		return nil, fmt.Errorf("failed to find submit button: %w", err)
	}

	log.Println("Clicking submit...")
	submitBtn.MustClick()

	log.Println("Waiting for results...")
	time.Sleep(5 * time.Second)

	result := &TrackingResult{
		CPF:       cpf,
		ScrapedAt: time.Now(),
		Events:    []TrackingEvent{},
	}

	html, err := page.HTML()
	if err != nil {
		return nil, fmt.Errorf("failed to get page HTML: %w", err)
	}

	log.Printf("Got HTML length: %d", len(html))

	htmlLower := strings.ToLower(html)

	if strings.Contains(htmlLower, "não encontrado") || strings.Contains(htmlLower, "not found") {
		result.Status = "não encontrado"
		return result, nil
	}

	if strings.Contains(htmlLower, "entregue") {
		result.Status = "entregue"
	} else if strings.Contains(htmlLower, "em trânsito") || strings.Contains(htmlLower, "transito") {
		result.Status = "em trânsito"
	} else if strings.Contains(htmlLower, "postado") {
		result.Status = "postado"
	} else {
		result.Status = "dados obtidos"
	}

	log.Printf("Tracking complete. Status: %s", result.Status)
	return result, nil
}

func (r *RodScraper) Close() error {
	if r.browser != nil {
		return r.browser.Close()
	}
	return nil
}
