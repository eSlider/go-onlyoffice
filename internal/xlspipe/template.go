package xlspipe

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

// Template names for oo docs put-xlsx --template.
const (
	TemplateCutoverPortugal = "cutover-portugal"
)

// BuildTemplate returns an xlsx workbook for a known template name.
func BuildTemplate(name string) (*File, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case TemplateCutoverPortugal, "portugal-cutover", "cutover":
		return BuildCutoverPortugalWorkbook(DefaultCutoverPortugal())
	default:
		return nil, fmt.Errorf("unknown xlsx template %q (try: %s)", name, TemplateCutoverPortugal)
	}
}

// File is an alias so cmd/oo can refer to excelize.File without importing excelize in every handler.
type File = excelize.File
