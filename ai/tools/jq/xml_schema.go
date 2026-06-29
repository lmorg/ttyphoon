package jq

import (
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
)

type jqXMLInput struct {
	XMLName xml.Name `xml:"xml"`
	Query   string   `xml:"query"`
	JSON    string   `xml:"json"`
}

func parseJqXMLInput(input string) (*jqXMLInput, error) {
	var parsed jqXMLInput

	decoder := xml.NewDecoder(strings.NewReader(input))
	if err := decoder.Decode(&parsed); err != nil {
		return nil, fmt.Errorf("failed to parse jq XML input: %w", err)
	}

	parsed.Query = strings.TrimSpace(parsed.Query)
	parsed.JSON = strings.TrimSpace(parsed.JSON)

	if parsed.Query == "" {
		return nil, errors.New("invalid jq XML input: missing <query>")
	}
	if parsed.JSON == "" {
		return nil, errors.New("invalid jq XML input: missing <json>")
	}

	return &parsed, nil
}
