package aiassistant

// ExportParseResponse expose parseResponse aux tests externes.
// Fichier uniquement présent lors des tests (nom export_test.go).
func ExportParseResponse(raw string) QueryResult {
	return parseResponse(raw)
}