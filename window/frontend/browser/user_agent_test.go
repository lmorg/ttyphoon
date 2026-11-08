package browser

import "testing"

func TestUserAgentFromRelease(t *testing.T) {
	got, err := userAgentFromRelease("26.6.2\n")
	if err != nil {
		t.Fatal(err)
	}
	if want := "Version/26.6.2 Safari/605.1.15"; got != want {
		t.Fatalf("user agent = %q, want %q", got, want)
	}
	for _, output := range []string{"", "\n", "26.6.2\nunexpected", "26 6 2"} {
		if _, err := userAgentFromRelease(output); err == nil {
			t.Errorf("expected error for %q", output)
		}
	}
}
