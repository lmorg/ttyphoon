package iterm2

/*
	Writing parsers is boring and I don't particularly enjoy working with XML
	either. So the following code was written by ChatGPT
*/

import (
	"encoding/xml"
	"fmt"
	"io"
	"strconv"

	"github.com/lmorg/mxtty/debug"
)

// Plist represents the root structure of the plist file
type Plist struct {
	XMLName xml.Name `xml:"plist"`
	Dict    Dict     `xml:"dict"`
}

// Dict holds the parsed color mappings
type Dict struct {
	Colors map[string]Color
}

// Color represents an RGB color
type Color struct {
	Red   float64
	Green float64
	Blue  float64
	Alpha float64
}

func unmarshalTheme(reader io.Reader) (map[string]Color, error) {
	// Decode XML
	var plist Plist

	decoder := xml.NewDecoder(reader)
	if err := decoder.Decode(&plist); err != nil {
		return nil, fmt.Errorf("error decoding XML: %v", err)
	}

	debug.Log(plist.Dict.Colors)

	return plist.Dict.Colors, nil
}

// UnmarshalXML manually parses the XML while differentiating between theme names and RGB components
func (d *Dict) UnmarshalXML(decoder *xml.Decoder, start xml.StartElement) error {
	d.Colors = make(map[string]Color)
	var key string
	colorMap := make(map[string]Color) // Temporary map to store colors
	var colorKey string                // The name of the color (e.g., "Ansi 0 Color")
	var currentColor *Color

	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch tok := tok.(type) {
		case xml.StartElement:
			switch tok.Name.Local {
			case "key":
				var keyName string
				if err := decoder.DecodeElement(&keyName, &tok); err != nil {
					return err
				}
				key = keyName

				// If we encounter a color key, we prepare a new Color struct
				if keyName != "Red Component" && keyName != "Green Component" && keyName != "Blue Component" && keyName != "Alpha Component" && keyName != "Color Space" {
					colorKey = keyName      // Store color name (e.g., "Ansi 1 Color")
					currentColor = &Color{} // Prepare a new color struct
				}
			case "real":
				var valueStr string
				if err := decoder.DecodeElement(&valueStr, &tok); err != nil {
					return err
				}
				value, err := strconv.ParseFloat(valueStr, 64)
				if err != nil {
					return err
				}

				// Assign the value to the correct RGB component
				if currentColor != nil {
					switch key {
					case "Red Component":
						currentColor.Red = value
					case "Green Component":
						currentColor.Green = value
					case "Blue Component":
						currentColor.Blue = value
					case "Alpha Component":
						currentColor.Alpha = value
					}
				}
			}
		case xml.EndElement:
			if tok.Name.Local == "dict" && currentColor != nil {
				colorMap[colorKey] = *currentColor // Store the fully parsed color
				currentColor = nil                 // Reset for next color
			}
		}
	}

	d.Colors = colorMap
	return nil
}
