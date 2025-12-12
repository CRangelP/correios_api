package scraper

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

type RodScraper struct {
	browserURL     string
	localPath      string
	isRemote       bool
	browser        *rod.Browser
	browserMu      sync.Mutex
	semaphore      chan struct{}
	maxConcurrency int
}

func getWebSocketURL(baseURL string) (string, error) {
	httpURL := strings.Replace(baseURL, "ws://", "http://", 1)
	httpURL = strings.Replace(httpURL, "wss://", "https://", 1)
	versionURL := httpURL + "/json/version"
	
	resp, err := http.Get(versionURL)
	if err != nil {
		return "", fmt.Errorf("failed to get version info: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}
	
	var versionInfo struct {
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}
	
	if err := json.Unmarshal(body, &versionInfo); err != nil {
		return "", fmt.Errorf("failed to parse version info: %w", err)
	}
	
	if versionInfo.WebSocketDebuggerURL == "" {
		return "", fmt.Errorf("webSocketDebuggerUrl not found in response")
	}
	
	return versionInfo.WebSocketDebuggerURL, nil
}

func NewRodScraper(browserURL string, maxConcurrency int) (*RodScraper, error) {
	if maxConcurrency <= 0 {
		maxConcurrency = 3
	}

	scraper := &RodScraper{
		maxConcurrency: maxConcurrency,
		semaphore:      make(chan struct{}, maxConcurrency),
	}

	if browserURL != "" {
		log.Printf("Configured remote browser at: %s (max concurrency: %d)", browserURL, maxConcurrency)
		scraper.browserURL = browserURL
		scraper.isRemote = true
	} else {
		path, found := launcher.LookPath()
		if !found {
			path = "/usr/bin/chromium"
		}
		log.Printf("Configured local browser at: %s (max concurrency: %d)", path, maxConcurrency)
		scraper.localPath = path
		scraper.isRemote = false
	}

	if err := scraper.ensureBrowser(); err != nil {
		return nil, fmt.Errorf("failed to initialize browser: %w", err)
	}

	return scraper, nil
}

func (r *RodScraper) connectBrowser() (*rod.Browser, error) {
	var browser *rod.Browser
	var err error

	if r.isRemote {
		log.Printf("Connecting to remote browser at: %s", r.browserURL)
		
		var wsURL string
		maxRetries := 10
		for i := 0; i < maxRetries; i++ {
			wsURL, err = getWebSocketURL(r.browserURL)
			if err == nil {
				log.Printf("Got WebSocket URL: %s", wsURL)
				break
			}
			log.Printf("Attempt %d/%d: Failed to get WebSocket URL: %v. Retrying in 2s...", i+1, maxRetries, err)
			time.Sleep(2 * time.Second)
		}
		
		if err != nil {
			return nil, fmt.Errorf("failed to get WebSocket URL after %d attempts: %w", maxRetries, err)
		}
		
		browser = rod.New().ControlURL(wsURL)
		if err = browser.Connect(); err != nil {
			return nil, fmt.Errorf("failed to connect to browser: %w", err)
		}
		log.Printf("Connected to remote browser successfully")
	} else {
		log.Printf("Launching local browser at: %s", r.localPath)
		u := launcher.New().Bin(r.localPath).Headless(true).NoSandbox(true).MustLaunch()
		browser = rod.New().ControlURL(u).MustConnect()
	}

	return browser, nil
}

func (r *RodScraper) ensureBrowser() error {
	r.browserMu.Lock()
	defer r.browserMu.Unlock()

	if r.browser != nil {
		return nil
	}

	browser, err := r.connectBrowser()
	if err != nil {
		return err
	}
	r.browser = browser
	return nil
}

func (r *RodScraper) reconnectBrowser() error {
	r.browserMu.Lock()
	defer r.browserMu.Unlock()

	if r.browser != nil {
		_ = r.browser.Close()
		r.browser = nil
	}

	log.Println("Reconnecting to browser...")
	browser, err := r.connectBrowser()
	if err != nil {
		return err
	}
	r.browser = browser
	log.Println("Browser reconnected successfully")
	return nil
}

func (r *RodScraper) getPage() (*rod.Page, error) {
	r.browserMu.Lock()
	defer r.browserMu.Unlock()

	if r.browser == nil {
		return nil, fmt.Errorf("browser not connected")
	}

	page, err := r.browser.Page(proto.TargetCreateTarget{URL: ""})
	if err != nil {
		return nil, err
	}
	return page, nil
}

func (r *RodScraper) TrackCPF(cpf string) (*TrackingResult, error) {
	r.semaphore <- struct{}{}
	defer func() { <-r.semaphore }()

	log.Printf("Starting tracking for CPF: %s", cpf)

	if err := r.ensureBrowser(); err != nil {
		return nil, fmt.Errorf("failed to ensure browser: %w", err)
	}

	page, err := r.getPage()
	if err != nil {
		log.Printf("Failed to get page, attempting reconnect: %v", err)
		if reconnErr := r.reconnectBrowser(); reconnErr != nil {
			return nil, fmt.Errorf("failed to reconnect browser: %w", reconnErr)
		}
		page, err = r.getPage()
		if err != nil {
			return nil, fmt.Errorf("failed to get page after reconnect: %w", err)
		}
	}

	page.MustSetUserAgent(&proto.NetworkSetUserAgentOverride{
		UserAgent: "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	})

	defer func() {
		if err := page.Close(); err != nil {
			log.Printf("Error closing page: %v", err)
		}
	}()

	page = page.Timeout(60 * time.Second)

	if err := page.Navigate("https://www.haga7digital.com.br/?page=rastreio"); err != nil {
		if strings.Contains(err.Error(), "closed") {
			log.Printf("Connection closed during navigate, triggering reconnect")
			_ = r.reconnectBrowser()
		}
		return nil, fmt.Errorf("failed to navigate: %w", err)
	}

	log.Println("Page navigated, waiting for load...")
	if err := page.WaitLoad(); err != nil {
		return nil, fmt.Errorf("failed to wait for load: %w", err)
	}
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
	time.Sleep(8 * time.Second)

	result := &TrackingResult{
		CPF:       cpf,
		ScrapedAt: time.Now(),
		Events:    []TrackingEvent{},
	}

	if err := page.WaitLoad(); err != nil {
		log.Printf("Warning: WaitLoad failed: %v", err)
	}

	pageText, err := page.Eval(`() => document.body.innerText`)
	if err != nil {
		return nil, fmt.Errorf("failed to get page text: %w", err)
	}

	text := pageText.Value.String()
	log.Printf("Got page text length: %d", len(text))

	if strings.Contains(strings.ToLower(text), "não encontrado") || strings.Contains(strings.ToLower(text), "not found") {
		result.Status = "não encontrado"
		return result, nil
	}

	trackingCodeRegex := regexp.MustCompile(`([A-Z]{2}\d{9}[A-Z]{2})\s*-\s*(\w+)`)
	if matches := trackingCodeRegex.FindStringSubmatch(text); len(matches) >= 2 {
		result.TrackingCode = matches[1]
		if len(matches) >= 3 {
			result.TrackingCode = matches[1] + " - " + matches[2]
		}
	}

	expectedDateRegex := regexp.MustCompile(`Data prevista:\s*(\d{2}/\d{2}/\d{4})`)
	if matches := expectedDateRegex.FindStringSubmatch(text); len(matches) >= 2 {
		result.ExpectedDate = matches[1]
	}

	lines := strings.Split(text, "\n")
	var currentEvent TrackingEvent
	dateTimeRegex := regexp.MustCompile(`(\d{2}/\d{2}/\d{4}\s+\d{2}:\d{2}:\d{2})`)

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.Contains(line, "Objeto em transferência") ||
			strings.Contains(line, "Objeto postado") ||
			strings.Contains(line, "Objeto entregue") ||
			strings.Contains(line, "Objeto não entregue") ||
			strings.Contains(line, "Etiqueta emitida") ||
			strings.Contains(line, "Objeto saiu") ||
			strings.Contains(line, "Objeto recebido") ||
			strings.Contains(line, "Objeto aguardando retirada") ||
			strings.Contains(line, "Objeto devolvido") ||
			strings.Contains(line, "Objeto encaminhado") ||
			strings.Contains(line, "Objeto retido") ||
			strings.Contains(line, "Objeto roubado") ||
			strings.Contains(line, "Objeto extraviado") ||
			strings.Contains(line, "Objeto avariado") ||
			strings.Contains(line, "Fiscalização aduaneira") ||
			strings.Contains(line, "Aguardando pagamento") ||
			strings.Contains(line, "Pagamento confirmado") ||
			strings.Contains(line, "Tentativa de entrega") ||
			strings.Contains(line, "Saída para entrega") ||
			strings.Contains(line, "Objeto coletado") ||
			strings.Contains(line, "Coleta solicitada") ||
			strings.Contains(line, "Logística reversa") ||
			strings.Contains(line, "Destinatário ausente") ||
			strings.Contains(line, "Endereço incorreto") ||
			strings.Contains(line, "Endereço insuficiente") ||
			strings.Contains(line, "Objeto em trânsito") ||
			strings.Contains(line, "Objeto disponível") {
			
			if currentEvent.Description != "" {
				result.Events = append(result.Events, currentEvent)
			}
			currentEvent = TrackingEvent{
				Description: line,
			}
		} else if strings.HasPrefix(line, "Unidade de Tratamento") ||
			strings.HasPrefix(line, "Unidade de Distribuição") ||
			strings.HasPrefix(line, "Postado") {
			currentEvent.LocationType = line
		} else if dateTimeRegex.MatchString(line) {
			currentEvent.Date = line
		} else if len(line) > 2 && strings.Contains(line, ",") && !strings.Contains(line, "CORREIOS") && !strings.Contains(line, "HAGA") {
			if i > 0 && (strings.HasPrefix(lines[i-1], "Unidade") || strings.HasPrefix(lines[i-1], "Postado")) {
				currentEvent.Location = line
			}
		} else if regexp.MustCompile(`^[A-Z\s]+,[A-Z]{2}$`).MatchString(line) {
			currentEvent.Location = line
		}
	}

	if currentEvent.Description != "" {
		result.Events = append(result.Events, currentEvent)
	}

	if len(result.Events) > 0 {
		firstEvent := result.Events[0]
		desc := strings.ToLower(firstEvent.Description)
		if strings.Contains(desc, "entregue ao destinatário") || strings.Contains(desc, "objeto entregue") {
			result.Status = "entregue"
		} else if strings.Contains(desc, "não entregue") || strings.Contains(desc, "destinatário ausente") {
			result.Status = "tentativa de entrega"
		} else if strings.Contains(desc, "saiu para entrega") || strings.Contains(desc, "saída para entrega") {
			result.Status = "saiu para entrega"
		} else if strings.Contains(desc, "aguardando retirada") {
			result.Status = "aguardando retirada"
		} else if strings.Contains(desc, "devolvido") {
			result.Status = "devolvido"
		} else if strings.Contains(desc, "retido") || strings.Contains(desc, "fiscalização") {
			result.Status = "retido na fiscalização"
		} else if strings.Contains(desc, "extraviado") || strings.Contains(desc, "roubado") {
			result.Status = "extraviado"
		} else if strings.Contains(desc, "avariado") {
			result.Status = "avariado"
		} else if strings.Contains(desc, "aguardando pagamento") {
			result.Status = "aguardando pagamento"
		} else if strings.Contains(desc, "transferência") || strings.Contains(desc, "trânsito") || strings.Contains(desc, "encaminhado") {
			result.Status = "em trânsito"
		} else if strings.Contains(desc, "postado") || strings.Contains(desc, "coletado") {
			result.Status = "postado"
		} else if strings.Contains(desc, "etiqueta") {
			result.Status = "etiqueta emitida"
		} else {
			result.Status = "em processamento"
		}
	} else {
		result.Status = "dados obtidos"
	}

	log.Printf("Tracking complete. Status: %s, Events: %d", result.Status, len(result.Events))
	return result, nil
}

func (r *RodScraper) Close() error {
	r.browserMu.Lock()
	defer r.browserMu.Unlock()

	if r.browser != nil {
		if err := r.browser.Close(); err != nil {
			return fmt.Errorf("failed to close browser: %w", err)
		}
		r.browser = nil
	}
	return nil
}
