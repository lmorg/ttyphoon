package websearch

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"
)

func (t *DuckDuckGoSearch) cachedFilterAdvice(input *duckDuckGoInputT) ([]string, []string, error) {
	resolved := slices.Clone(input.Engines)
	t.discoveryMu.Lock()
	data, expires := t.discoveryData, t.discoveryExpires
	t.discoveryMu.Unlock()
	if data == "" || !time.Now().Before(expires) {
		return resolved, nil, nil
	}
	var discovery discoveryResult
	if json.Unmarshal([]byte(data), &discovery) != nil {
		return resolved, nil, nil
	}
	var warnings []string
	warn := func(message string) {
		if len(warnings) < 6 && !slices.Contains(warnings, message) {
			warnings = append(warnings, message)
		}
	}
	for _, category := range input.Categories {
		known := slices.Contains(discovery.Categories, category)
		for _, engine := range discovery.Engines {
			known = known || slices.Contains(engine.Categories, category)
		}
		if !known {
			if !discovery.Truncated {
				return nil, nil, fmt.Errorf("unknown SearXNG category %q in cached discovery; use action=discover with refresh=true to reload names", category)
			}
			warn("Cached discovery is truncated; some category names could not be verified.")
		}
	}
	for index, name := range input.Engines {
		known := false
		for _, engine := range discovery.Engines {
			if name == engine.Name || (engine.Shortcut != "" && strings.TrimPrefix(name, "!") == engine.Shortcut) {
				resolved[index] = engine.Name
				known = true
				break
			}
		}
		if !known {
			if !discovery.Truncated {
				return nil, nil, fmt.Errorf("unknown SearXNG engine %q in cached discovery; use action=discover with refresh=true to reload names", name)
			}
			warn("Cached discovery is truncated; some engine names could not be verified.")
		}
	}
	for _, engine := range discovery.Engines {
		explicit := slices.Contains(resolved, engine.Name)
		if len(resolved) > 0 && !explicit {
			continue
		}
		if !explicit && engine.Enabled != nil && !*engine.Enabled {
			continue
		}
		if len(input.Categories) > 0 {
			matches := false
			for _, category := range input.Categories {
				matches = matches || slices.Contains(engine.Categories, category)
			}
			if !matches {
				if explicit {
					warn(fmt.Sprintf("Engine %s advertises different categories; the instance may ignore this combination.", searchResultText(engine.Name, 120)))
				}
				continue
			}
		}
		name := searchResultText(engine.Name, 120)
		if explicit && engine.Enabled != nil && !*engine.Enabled {
			warn(fmt.Sprintf("Engine %s is disabled by default; explicit selection is being attempted.", name))
		}
		if input.Page > 1 && engine.Paging != nil && !*engine.Paging {
			warn(fmt.Sprintf("Engine %s does not advertise pagination; later pages may be empty or repeated.", name))
		}
		if input.TimeRange != "" && engine.TimeRangeSupport != nil && !*engine.TimeRangeSupport {
			warn(fmt.Sprintf("Engine %s does not advertise time-range support; the filter may be ignored.", name))
		}
		if input.Language != "" && engine.LanguageSupport != nil && !*engine.LanguageSupport {
			warn(fmt.Sprintf("Engine %s does not advertise language filtering; the filter may be ignored.", name))
		}
	}
	return resolved, warnings, nil
}
