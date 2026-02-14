package detector

// Rule defines the interface for detection rules
type Rule interface {
	// Name returns the name of the rule
	Name() string
	
	// Analyze checks the input text and returns findings if any
	Analyze(text string) []Finding
}

