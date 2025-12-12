package scraper

type Scraper interface {
	TrackCPF(cpf string) (*TrackingResult, error)
	Close() error
}
