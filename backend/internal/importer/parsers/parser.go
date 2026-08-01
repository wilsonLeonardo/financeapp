package parsers

import (
	"encoding/csv"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/financeapp/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type ParsedTransaction struct {
	Description string
	Amount      decimal.Decimal
	Type        domain.TransactionType
	Date        time.Time
}

func ParseCSV(r io.Reader, userID uuid.UUID, importID string) ([]*domain.Expense, []string) {
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, []string{"failed to read file"}
	}

	sep := detectSeparator(string(content))
	reader := csv.NewReader(strings.NewReader(string(content)))
	reader.Comma = sep
	reader.TrimLeadingSpace = true
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1

	records, err := reader.ReadAll()
	if err != nil {
		return nil, []string{fmt.Sprintf("failed to parse CSV: %v", err)}
	}
	if len(records) < 2 {
		return nil, []string{"CSV has no data rows"}
	}

	colIdx := map[string]int{}
	for i, h := range records[0] {
		colIdx[strings.TrimSpace(strings.ToLower(h))] = i
	}

	dateCol := resolveCol(colIdx, "data de compra", "data compra", "data", "date", "dt")
	descCol := resolveCol(colIdx, "descrição", "descricao", "description", "historico", "histórico", "memo", "nome no cartão", "nome")
	amountCol := resolveCol(colIdx, "valor (em $)", "valor em $", "valor ($)", "valor", "amount", "value", "vlr")

	if dateCol == -1 {
		dateCol = 0
	}
	if descCol == -1 {
		descCol = 1
	}
	if amountCol == -1 {
		amountCol = 2
	}

	maxCol := maxInt(dateCol, descCol, amountCol)

	var expenses []*domain.Expense
	var errors []string

	for i, record := range records {
		if i == 0 {
			continue
		}
		if len(record) <= maxCol {
			errors = append(errors, fmt.Sprintf("row %d: insufficient columns", i+1))
			continue
		}

		date, err := parseDate(strings.TrimSpace(record[dateCol]))
		if err != nil {
			errors = append(errors, fmt.Sprintf("row %d: invalid date %q", i+1, record[dateCol]))
			continue
		}

		description := strings.TrimSpace(record[descCol])
		if description == "" {
			description = "Sem descrição"
		}

		amount, err := parseBRAmount(strings.TrimSpace(record[amountCol]))
		if err != nil {
			errors = append(errors, fmt.Sprintf("row %d: invalid amount %q", i+1, record[amountCol]))
			continue
		}
		if amount.IsZero() {
			continue
		}

		txType := domain.TransactionTypeExpense
		if amount.IsNegative() {
			txType = domain.TransactionTypeIncome
		}
		amount = amount.Abs()

		importRef := fmt.Sprintf("%s-csv-%s-%d", importID, date.Format("20060102"), i)
		expenses = append(expenses, &domain.Expense{
			UserID:      userID,
			Amount:      amount,
			Type:        txType,
			Description: description,
			Date:        date,
			ImportID:    &importRef,
		})
	}

	return expenses, errors
}

// parseBRAmount parses monetary values in Brazilian or US format.
//
//	BR:    1.234,56  →  1234.56
//	US:    1,234.56  →  1234.56
//	Plain: 98.00     →  98.00   (2 decimal digits → decimal separator)
//	Plain: 1.000     →  1000    (3 decimal digits → thousands separator)
func parseBRAmount(s string) (decimal.Decimal, error) {
	s = strings.ReplaceAll(s, "$", "")
	s = strings.ReplaceAll(s, "$", "")
	s = strings.TrimSpace(s)

	negative := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	s = strings.TrimPrefix(s, "+")

	if s == "" {
		return decimal.Zero, fmt.Errorf("empty amount")
	}

	hasDot := strings.Contains(s, ".")
	hasComma := strings.Contains(s, ",")

	var normalized string

	switch {
	case hasDot && hasComma:
		dotPos := strings.LastIndex(s, ".")
		commaPos := strings.LastIndex(s, ",")
		if commaPos > dotPos {
			// BR format: 1.234,56
			s = strings.ReplaceAll(s, ".", "")
			normalized = strings.ReplaceAll(s, ",", ".")
		} else {
			// US format: 1,234.56
			normalized = strings.ReplaceAll(s, ",", "")
		}

	case hasComma && !hasDot:
		parts := strings.Split(s, ",")
		last := parts[len(parts)-1]
		if len(parts) == 2 && (len(last) == 1 || len(last) == 2) {
			// Decimal comma: 98,00
			normalized = strings.ReplaceAll(s, ",", ".")
		} else {
			// Thousands comma: 1,000
			normalized = strings.ReplaceAll(s, ",", "")
		}

	case hasDot && !hasComma:
		parts := strings.Split(s, ".")
		last := parts[len(parts)-1]
		if len(parts) == 2 && len(last) == 3 {
			// Thousands dot: 1.000
			normalized = strings.ReplaceAll(s, ".", "")
		} else {
			// Decimal dot: 98.00
			normalized = s
		}

	default:
		normalized = regexp.MustCompile(`[^\d]`).ReplaceAllString(s, "")
	}

	amount, err := decimal.NewFromString(normalized)
	if err != nil {
		return decimal.Zero, fmt.Errorf("cannot parse %q", s)
	}
	if negative {
		amount = amount.Neg()
	}
	return amount, nil
}

func ParseOFX(r io.Reader, userID uuid.UUID, importID string) ([]*domain.Expense, []string) {
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, []string{"failed to read OFX file"}
	}

	var expenses []*domain.Expense
	var errors []string
	lines := strings.Split(string(content), "\n")
	var currentTx *ParsedTransaction

	for i, line := range lines {
		line = strings.TrimSpace(line)
		switch {
		case line == "<STMTTRN>":
			currentTx = &ParsedTransaction{}
		case line == "</STMTTRN>" && currentTx != nil:
			if currentTx.Description != "" && !currentTx.Amount.IsZero() {
				importRef := fmt.Sprintf("%s-ofx-%s-%d", importID, currentTx.Date.Format("20060102"), i)
				expenses = append(expenses, &domain.Expense{
					UserID:      userID,
					Amount:      currentTx.Amount.Abs(),
					Type:        currentTx.Type,
					Description: currentTx.Description,
					Date:        currentTx.Date,
					ImportID:    &importRef,
				})
			}
			currentTx = nil
		case currentTx != nil && strings.HasPrefix(line, "<TRNAMT>"):
			val := extractOFXValue(line)
			amount, err := decimal.NewFromString(val)
			if err != nil {
				errors = append(errors, fmt.Sprintf("line %d: invalid amount", i+1))
				continue
			}
			currentTx.Amount = amount
			if amount.IsNegative() {
				currentTx.Type = domain.TransactionTypeExpense
			} else {
				currentTx.Type = domain.TransactionTypeIncome
			}
		case currentTx != nil && strings.HasPrefix(line, "<MEMO>"):
			currentTx.Description = extractOFXValue(line)
		case currentTx != nil && strings.HasPrefix(line, "<NAME>"):
			if currentTx.Description == "" {
				currentTx.Description = extractOFXValue(line)
			}
		case currentTx != nil && strings.HasPrefix(line, "<DTPOSTED>"):
			val := extractOFXValue(line)
			if len(val) >= 8 {
				val = val[:8]
			}
			if t, err := time.Parse("20060102", val); err == nil {
				currentTx.Date = t
			}
		}
	}

	return expenses, errors
}

func extractOFXValue(line string) string {
	start := strings.Index(line, ">")
	if start == -1 {
		return ""
	}
	val := line[start+1:]
	if end := strings.Index(val, "<"); end != -1 {
		val = val[:end]
	}
	return strings.TrimSpace(val)
}

func parseDate(s string) (time.Time, error) {
	formats := []string{
		"2006-01-02", "02/01/2006", "01/02/2006",
		"02-01-2006", "2006/01/02", "20060102",
		"2/1/2006", "1/2/2006",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognised date format: %s", s)
}

func detectSeparator(content string) rune {
	firstLine := strings.SplitN(content, "\n", 2)[0]
	counts := map[rune]int{
		';':  strings.Count(firstLine, ";"),
		',':  strings.Count(firstLine, ","),
		'\t': strings.Count(firstLine, "\t"),
		'|':  strings.Count(firstLine, "|"),
	}
	sep := ','
	best := 0
	for r, c := range counts {
		if c > best {
			best = c
			sep = r
		}
	}
	return sep
}

func resolveCol(colIdx map[string]int, candidates ...string) int {
	for _, c := range candidates {
		if idx, ok := colIdx[strings.ToLower(c)]; ok {
			return idx
		}
	}
	return -1
}

func DetectFileType(filename string, r io.ReadSeeker) string {
	ext := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(ext, ".ofx"), strings.HasSuffix(ext, ".qfx"):
		return "ofx"
	case strings.HasSuffix(ext, ".csv"):
		return "csv"
	}
	buf := make([]byte, 512)
	n, _ := r.Read(buf)
	r.Seek(0, io.SeekStart)
	c := strings.ToUpper(string(buf[:n]))
	if strings.Contains(c, "OFXHEADER") || strings.Contains(c, "<OFX>") {
		return "ofx"
	}
	return "csv"
}

func maxInt(vals ...int) int {
	m := vals[0]
	for _, v := range vals[1:] {
		if v > m {
			m = v
		}
	}
	return m
}
