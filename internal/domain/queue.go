package domain

type Queue interface {
	Publish(urlGroup string, body map[string]any) error
}
