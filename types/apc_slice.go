package types

import (
	"encoding/json"
	"log"
	"strings"
)

type ApcSlice struct {
	slice []string
}

func NewApcSlice(apc []rune) *ApcSlice {
	s := string(apc)
	as := new(ApcSlice)

	slice := strings.Split(s, ";")
	if len(slice) > 3 {
		as.slice = slice[:2]
		l := len(slice[0]) + len(slice[1]) + 2
		as.slice = append(as.slice, s[l:])
	} else {
		as.slice = slice
	}

	//panic("^^")

	return as
}

func NewApcSliceNoParse(s []string) *ApcSlice {
	return &ApcSlice{s}
}

func (as *ApcSlice) Index(i int) string {
	if len(as.slice) <= i {
		return ""
	}
	return as.slice[i]
}

func (as *ApcSlice) Parameters(params any) error {
	s := as.Index(2)

	if s != "" {
		err := json.Unmarshal([]byte(s), params)
		if err != nil {
			log.Printf("WARNING: cannot decode APC string '%s': %v", s, err)
			return err
		}
	}

	return nil
}
