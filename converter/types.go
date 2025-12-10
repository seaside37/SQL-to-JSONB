package converter

type SQLMapper struct {
	OriginalSQL   string
	MappedSQL     string
	NumericFields map[string]struct{}
	AllFields     map[string][]string
	TableName     string
	PayloadCol    string
	Topic         string
}
