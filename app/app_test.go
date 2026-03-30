package app_test

import (
	"testing"

	"github.com/lmorg/ttyphoon/app"
)

func TestAppName(t *testing.T) {
	if app.Name() == "" {
		t.Error("Name isn't valid:")
		t.Log("  Name:", app.Name())
	}
}

/*
func TestVersion(t *testing.T) {
	rx := regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+( \(([-._/a-zA-Z0-9]+)\))?$`)

	if !rx.MatchString(app.Version()) {
		t.Error("Release version doesn't contain a valid string:")
		t.Log("  Version:", app.Version())
	}
}

func TestCopyright(t *testing.T) {
	rx := regexp.MustCompile(fmt.Sprintf(`^[0-9]{4}-%s .*$`, time.Now().Format("2006")))

	if !rx.MatchString(app.Copyright()) {
		t.Error("Copyright string doesn't contain a valid string (possibly out of date?):")
		t.Log("  Copyright:", app.Copyright())
	}
}

func TestLicense(t *testing.T) {
	if app.License() == "" {
		t.Error("License isn't valid:")
		t.Log("  License:", app.License())
	}
}
*/
