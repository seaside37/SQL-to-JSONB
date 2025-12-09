package converter

type SQLMapper struct {
	OriginalSQL   string
	MappedSQL     string
	NumericFields map[string]struct{}
}
