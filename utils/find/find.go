package find

import (
	"path"
	"regexp"
	"strings"
)

type FindT interface {
	MatchString(string) bool
	Filter([]string) []string
}

type stubFind struct{}

func (sf *stubFind) MatchString(string) bool {
	return false
}

func (sf *stubFind) Filter([]string) []string {
	return nil
}

func filterList(f FindT, list []string) []string {
	var results []string
	for i := range list {
		if f.MatchString(list[i]) {
			results = append(results, list[i])
		}
	}
	return results
}

type fuzzyFindT struct {
	stubFind
	mode   int
	tokens []string
}

func (ff *fuzzyFindT) Filter(list []string) []string { return filterList(ff, list) }

const (
	ffMatchAll  = 0
	ffMatchSome = iota + 1
	ffMatchNone
	ffMatchRegexp
	ffMatchGlob
)

func (ff *fuzzyFindT) MatchString(item string) bool {
	switch ff.mode {

	case ffMatchSome:
		return ff.matchSome(item)

	case ffMatchNone:
		return ff.matchNone(item)

	default:
		return ff.matchAll(item)
	}
}

func (ff *fuzzyFindT) matchAll(item string) bool {
	if len(ff.tokens) == 0 {
		return true
	}

	for i := range ff.tokens {
		if !strings.Contains(strings.ToLower(item), ff.tokens[i]) {
			return false
		}
	}

	return true
}

func (ff *fuzzyFindT) matchSome(item string) bool {
	if len(ff.tokens) == 0 {
		return true
	}

	for i := range ff.tokens {
		if strings.Contains(strings.ToLower(item), ff.tokens[i]) {
			return true
		}
	}

	return false
}

func (ff *fuzzyFindT) matchNone(item string) bool {
	if len(ff.tokens) == 0 {
		return false
	}

	for i := range ff.tokens {
		if strings.Contains(strings.ToLower(item), ff.tokens[i]) {
			return false
		}
	}

	return true
}

type globFindT struct {
	stubFind
	pattern string
}

func (gf *globFindT) Filter(list []string) []string { return filterList(gf, list) }

func (gf *globFindT) MatchString(item string) bool {
	lower := strings.ToLower(item)
	if found, _ := path.Match(gf.pattern, lower); found {
		return true
	}
	found, _ := path.Match(gf.pattern, path.Base(lower))
	return found
}

type regexpFindT struct {
	stubFind
	rx *regexp.Regexp
}

func (rf *regexpFindT) Filter(list []string) []string { return filterList(rf, list) }

func (rf *regexpFindT) MatchString(item string) bool {
	return rf.rx.MatchString(item)
}

func newGlobFind(pattern string) (*globFindT, error) {
	gf := new(globFindT)
	gf.pattern = pattern
	return gf, nil
}

func New(pattern string) (FindT, error) {
	pattern = strings.ToLower(pattern)
	ff := new(fuzzyFindT)

	ff.tokens = strings.Split(pattern, " ")

	for {
		if len(ff.tokens) == 0 {
			return ff, nil
		}

		if ff.tokens[len(ff.tokens)-1] == "" {
			ff.tokens = ff.tokens[:len(ff.tokens)-1]
		} else {
			break
		}
	}

	switch ff.tokens[0] {
	case "or":
		ff.mode = ffMatchSome
		ff.tokens = ff.tokens[1:]

	case "!":
		ff.mode = ffMatchNone
		ff.tokens = ff.tokens[1:]

	case "rx":
		ff.mode = ffMatchRegexp
		pattern = strings.Join(ff.tokens[1:], " ")
		rx, err := regexp.Compile("(?i)" + pattern)
		find := &regexpFindT{rx: rx}
		return find, err

	case "g":
		ff.mode = ffMatchGlob
		pattern = strings.Join(ff.tokens[1:], " ")
		find, err := newGlobFind(pattern)
		return find, err

	default:
		if strings.Contains(pattern, "*") || strings.Contains(pattern, "?") {
			ff.mode = ffMatchGlob
			find, err := newGlobFind(pattern)
			return find, err
		}
	}

	return ff, nil
}
