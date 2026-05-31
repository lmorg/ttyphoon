package lsp

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/lmorg/ttyphoon/utils/jupyter"
)

// Resolver provides deterministic language-id resolution for extensions and aliases.
type Resolver struct {
	byExtension map[string]string
	byExtAll    map[string][]string
	byAlias     map[string]string
	warnings    []string
}

func NewResolverFromJupyter(bindings []*jupyter.LanguageBindingT) *Resolver {
	res := &Resolver{
		byExtension: make(map[string]string),
		byExtAll:    make(map[string][]string),
		byAlias:     make(map[string]string),
	}

	for _, binding := range bindings {
		if binding == nil {
			continue
		}

		canonical := canonicalLanguageID(binding)
		if canonical == "" {
			continue
		}

		ext := normalizeExtension(binding.FileExtension)
		if ext != "" {
			for _, candidate := range languageCandidates(binding) {
				if !slices.Contains(res.byExtAll[ext], candidate) {
					res.byExtAll[ext] = append(res.byExtAll[ext], candidate)
				}
			}

			if existing, ok := res.byExtension[ext]; ok && existing != canonical {
				res.warnings = append(res.warnings, "duplicate extension mapping: ."+ext+" -> "+existing+", "+canonical+" (keeping first)")
			} else if !ok {
				res.byExtension[ext] = canonical
			}
		}

		for _, alias := range binding.Aliases {
			normalized := normalizeLanguageID(alias)
			if normalized == "" {
				continue
			}

			if existing, ok := res.byAlias[normalized]; ok && existing != canonical {
				res.warnings = append(res.warnings, "duplicate alias mapping: "+normalized+" -> "+existing+", "+canonical+" (keeping first)")
				continue
			}

			if _, ok := res.byAlias[normalized]; !ok {
				res.byAlias[normalized] = canonical
			}
		}
	}

	return res
}

func (r *Resolver) LanguageIDForFile(path string) string {
	if r == nil {
		return ""
	}

	ext := normalizeExtension(strings.TrimPrefix(filepath.Ext(path), "."))
	if ext == "" {
		return ""
	}

	return r.byExtension[ext]
}

func (r *Resolver) LanguageIDsForFile(path string) []string {
	if r == nil {
		return nil
	}

	ext := normalizeExtension(strings.TrimPrefix(filepath.Ext(path), "."))
	if ext == "" {
		return nil
	}

	if ids, ok := r.byExtAll[ext]; ok {
		return slices.Clone(ids)
	}

	if id := r.byExtension[ext]; id != "" {
		return []string{id}
	}

	return nil
}

func (r *Resolver) LanguageIDForName(name string) string {
	if r == nil {
		return ""
	}

	if name = normalizeLanguageID(name); name == "" {
		return ""
	}

	if languageID, ok := r.byAlias[name]; ok {
		return languageID
	}

	if languageID, ok := r.byExtension[normalizeExtension(name)]; ok {
		return languageID
	}

	return ""
}

func (r *Resolver) Warnings() []string {
	if r == nil {
		return nil
	}

	return slices.Clone(r.warnings)
}

var defaultResolver = NewResolverFromJupyter(jupyter.Languages)

func ResolveLanguageIDForFile(path string) string {
	return defaultResolver.LanguageIDForFile(path)
}

func ResolveLanguageIDsForFile(path string) []string {
	return defaultResolver.LanguageIDsForFile(path)
}

func ResolveLanguageIDForName(name string) string {
	return defaultResolver.LanguageIDForName(name)
}

func ResolverWarnings() []string {
	return defaultResolver.Warnings()
}

func canonicalLanguageID(binding *jupyter.LanguageBindingT) string {
	for _, alias := range binding.Aliases {
		if normalized := normalizeLanguageID(alias); normalized != "" {
			return normalized
		}
	}

	return normalizeExtension(binding.FileExtension)
}

func normalizeExtension(ext string) string {
	ext = strings.TrimPrefix(strings.TrimSpace(strings.ToLower(ext)), ".")
	return ext
}

func normalizeLanguageID(languageID string) string {
	return strings.TrimSpace(strings.ToLower(languageID))
}

func languageCandidates(binding *jupyter.LanguageBindingT) []string {
	if binding == nil {
		return nil
	}

	var out []string
	for _, alias := range binding.Aliases {
		if normalized := normalizeLanguageID(alias); normalized != "" && !slices.Contains(out, normalized) {
			out = append(out, normalized)
		}
	}

	if len(out) == 0 {
		if canonical := normalizeExtension(binding.FileExtension); canonical != "" {
			out = append(out, canonical)
		}
	}

	return out
}
