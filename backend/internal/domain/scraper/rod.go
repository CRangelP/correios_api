package scraper

import (
	"fmt"
	"log"
	"regexp"
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
	var err error

	if browserURL != "" {
		log.Printf("Connecting to remote browser at: %s", browserURL)
		
		maxRetries := 30
		for i := 0; i < maxRetries; i++ {
			browser = rod.New().ControlURL(browserURL)
			err = browser.Connect()
			if err == nil {
				log.Printf("Connected to remote browser successfully")
				break
			}
			log.Printf("Attempt %d/%d: Failed to connect to browser: %v. Retrying in 2s...", i+1, maxRetries, err)
			time.Sleep(2 * time.Second)
		}
		
		if err != nil {
			return nil, fmt.Errorf("failed to connect to remote browser after %d attempts: %w", maxRetries, err)
		}
	} else {
		path, found := launcher.LookPath()
		if !found {
			path = "/usr/bin/chromium"
		}
		log.Printf("Using local browser at: %s", path)
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

	page.Timeout(60 * time.Second)

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
	time.Sleep(8 * time.Second)

	result := &TrackingResult{
		CPF:       cpf,
		ScrapedAt: time.Now(),
		Events:    []TrackingEvent{},
	}

	page.MustWaitLoad()

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
	if r.browser != nil {
		return r.browser.Close()
	}
	return nil
}
