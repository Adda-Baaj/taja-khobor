package domain

// Domain contains core models and interfaces.

type Article struct {
	ID          string
	Title       string
	URL         string
	Description string
	ImageURL    string
}
