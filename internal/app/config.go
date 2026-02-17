package app

type Config struct {
	AIEndpoint  string
	DatabaseUrl string
	DataRoot    string
	WebEndpoint string

	// Calculated
	InputPath    string
	FinishedPath string
	ThumbsPath   string
}
