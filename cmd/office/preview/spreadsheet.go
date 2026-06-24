package preview

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"

	xls "github.com/eslider/go-xls/v2"
)

// CSVToMarkdownTable parses CSV bytes into a GitHub pipe table markdown string.
func CSVToMarkdownTable(data []byte) (string, error) {
	tab, err := parseCSV(data)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := xls.WriteMarkdownTable(&buf, tab); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()) + "\n", nil
}

func parseCSV(data []byte) (xls.Table, error) {
	r := csv.NewReader(bytes.NewReader(data))
	r.TrimLeadingSpace = true
	records, err := r.ReadAll()
	if err != nil {
		return xls.Table{}, err
	}
	if len(records) == 0 {
		return xls.Table{}, fmt.Errorf("preview: empty csv")
	}
	return xls.Table{Columns: records[0], Rows: records[1:]}, nil
}

// JSONToMarkdown wraps JSON in a fenced code block for preview.
func JSONToMarkdown(data []byte) (string, error) {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return "", err
	}
	pretty, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("```json\n%s\n```\n", string(pretty)), nil
}

// FileBytesToMarkdown picks a preview format based on file extension.
func FileBytesToMarkdown(name string, data []byte) (string, error) {
	ext := strings.ToLower(name)
	switch {
	case strings.HasSuffix(ext, ".csv"):
		return CSVToMarkdownTable(data)
	case strings.HasSuffix(ext, ".json"):
		return JSONToMarkdown(data)
	case strings.HasSuffix(ext, ".md"), strings.HasSuffix(ext, ".txt"):
		return string(data), nil
	default:
		return fmt.Sprintf("```\n(binary or unsupported preview for %s, %d bytes)\n```\n", name, len(data)), nil
	}
}
