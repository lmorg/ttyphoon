package metamd

import (
	_ "embed"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"

	"github.com/lmorg/murex/utils/humannumbers"
)

// Values are the metadata fields rendered into the Notes Meta markdown template.
type Values struct {
	Filename   string
	FileType   string
	SizeHuman  string
	SizeBytes  int64
	PathFull   string
	UserOwner  string
	GroupOwner string
	UnixOctal  string
	UserACL    string
	GroupACL   string
	OtherACL   string
}

//go:embed meta.md
var metaTemplateText string

var metaTemplate = template.Must(template.New("meta-md").Parse(metaTemplateText))

func withDefault(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func aclString(mode os.FileMode, readBit, writeBit, execBit os.FileMode) string {
	chars := []byte{'-', '-', '-'}
	if mode&readBit != 0 {
		chars[0] = 'r'
	}
	if mode&writeBit != 0 {
		chars[1] = 'w'
	}
	if mode&execBit != 0 {
		chars[2] = 'x'
	}
	return string(chars)
}

func statOwnerGroupIDs(sys any) (string, string) {
	v := reflect.ValueOf(sys)
	if !v.IsValid() {
		return "", ""
	}

	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return "", ""
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return "", ""
	}

	uidField := v.FieldByName("Uid")
	gidField := v.FieldByName("Gid")

	uid := ""
	gid := ""

	if uidField.IsValid() {
		uid = fmt.Sprint(uidField.Interface())
	}
	if gidField.IsValid() {
		gid = fmt.Sprint(gidField.Interface())
	}

	return uid, gid
}

func lookupUserName(uid string) string {
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return "unknown"
	}

	u, err := user.LookupId(uid)
	if err != nil {
		return uid
	}

	if strings.TrimSpace(u.Username) != "" {
		return u.Username
	}

	return uid
}

func lookupGroupName(gid string) string {
	gid = strings.TrimSpace(gid)
	if gid == "" {
		return "unknown"
	}

	g, err := user.LookupGroupId(gid)
	if err != nil {
		return gid
	}

	if strings.TrimSpace(g.Name) != "" {
		return g.Name
	}

	return gid
}

// DocumentForPath returns a complete markdown metadata document for a file path.
func DocumentForPath(resolvedPath string) string {
	meta := Values{
		Filename:   filepath.Base(resolvedPath),
		FileType:   "unknown",
		SizeHuman:  "unknown",
		PathFull:   resolvedPath,
		UserOwner:  "unknown",
		GroupOwner: "unknown",
		UnixOctal:  "0000",
		UserACL:    "---",
		GroupACL:   "---",
		OtherACL:   "---",
	}

	fi, err := os.Stat(resolvedPath)
	if err != nil {
		return document(meta)
	}

	meta.Filename = fi.Name()
	meta.FileType = fileType(resolvedPath)
	meta.SizeBytes = fi.Size()
	meta.SizeHuman = humannumbers.Bytes(uint64(fi.Size()))
	meta.UnixOctal = fmt.Sprintf("%04o", fi.Mode().Perm())
	meta.UserACL = aclString(fi.Mode(), 0400, 0200, 0100)
	meta.GroupACL = aclString(fi.Mode(), 0040, 0020, 0010)
	meta.OtherACL = aclString(fi.Mode(), 0004, 0002, 0001)

	uid, gid := statOwnerGroupIDs(fi.Sys())
	meta.UserOwner = lookupUserName(uid)
	meta.GroupOwner = lookupGroupName(gid)

	return document(meta)
}

// Document returns a complete markdown document for the notes Meta tab.
func document(v Values) string {
	data := Values{
		Filename:   withDefault(v.Filename, "Unknown file"),
		FileType:   withDefault(v.FileType, "unknown"),
		SizeBytes:  v.SizeBytes,
		SizeHuman:  withDefault(v.SizeHuman, "unknown"),
		PathFull:   withDefault(v.PathFull, ""),
		UserOwner:  withDefault(v.UserOwner, "unknown"),
		GroupOwner: withDefault(v.GroupOwner, "unknown"),
		UnixOctal:  withDefault(v.UnixOctal, "0000"),
		UserACL:    withDefault(v.UserACL, "---"),
		GroupACL:   withDefault(v.GroupACL, "---"),
		OtherACL:   withDefault(v.OtherACL, "---"),
	}

	var b strings.Builder
	if err := metaTemplate.Execute(&b, data); err != nil {
		return "# " + data.Filename
	}

	return strings.TrimSpace(b.String())
}
