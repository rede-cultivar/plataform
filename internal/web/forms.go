package web

import (
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rede-cultivar/plataform/internal/cultivation"
	"github.com/rede-cultivar/plataform/internal/practices"
)

const (
	dateLayout     = "2006-01-02"
	dateTimeLayout = "2006-01-02T15:04"
)

func parseDate(value string) (time.Time, error) {
	return time.ParseInLocation(dateLayout, strings.TrimSpace(value), time.Local)
}

func parseDateTime(value string) (time.Time, error) {
	return time.ParseInLocation(dateTimeLayout, strings.TrimSpace(value), time.Local)
}

func parseOptionalDate(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := parseDate(value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func parseOptionalStage(value string) *cultivation.Stage {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	stage := cultivation.Stage(value)
	return &stage
}

func parseOptionalFloat(value string) (*float64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseFloat(strings.ReplaceAll(value, ",", "."), 64)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

// parseMeasurements lê linhas no formato "nome=valor" ou "nome: valor".
func parseMeasurements(raw string) map[string]string {
	result := make(map[string]string)

	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		index := strings.IndexAny(line, "=:")
		if index < 0 {
			result[line] = ""
			continue
		}

		key := strings.TrimSpace(line[:index])
		value := strings.TrimSpace(line[index+1:])
		if key != "" {
			result[key] = value
		}
	}

	return result
}

// parseList lê itens separados por vírgula ou quebra de linha.
func parseList(raw string) []string {
	raw = strings.ReplaceAll(raw, "\n", ",")
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

// parseLines lê uma lista com um item por linha.
func parseLines(raw string) []string {
	result := make([]string, 0)
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}

func linesText(items []string) string {
	return strings.Join(items, "\n")
}

func listText(items []string) string {
	return strings.Join(items, ", ")
}

func measurementsText(measurements map[string]string) string {
	keys := make([]string, 0, len(measurements))
	for key := range measurements {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, key+"="+measurements[key])
	}
	return strings.Join(lines, "\n")
}

func validationMessage(err error) (string, bool) {
	var cultivationError *cultivation.ValidationError
	if errors.As(err, &cultivationError) {
		return cultivationError.Message, true
	}

	var practiceError *practices.ValidationError
	if errors.As(err, &practiceError) {
		return practiceError.Message, true
	}

	return "", false
}
