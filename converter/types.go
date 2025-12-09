package converter

type SQLMapper struct {
	OriginalSQL   string
	MappedSQL     string
	NumericFields map[string]struct{}
	TableName     string
	PayloadCol    string
	Topic         string
}
