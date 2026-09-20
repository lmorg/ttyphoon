package sessiondb

import (
	"strings"
	"testing"
	"time"
)

func TestBuildSessionLogRequestPrefix_RendersUploadedImageAfterAttachment(t *testing.T) {
	prefix := buildSessionLogRequestPrefix(
		"",
		"generated-image-123.png",
		"Image attachment: generated-image-123.png\n\n![uploaded /home/user/ttyphoon/.images/generated-image-123.png](/home/user/ttyphoon/.images/generated-image-123.png)",
		time.Unix(0, 0),
	)

	if !strings.Contains(prefix, "~~~text\nImage attachment: generated-image-123.png\n\n~~~\n\n") {
		t.Fatalf("prefix does not contain the attachment output block:\n%s", prefix)
	}
	if !strings.Contains(prefix, "~~~\n\n![uploaded /home/user/ttyphoon/.images/generated-image-123.png](/home/user/ttyphoon/.images/generated-image-123.png)\n\n") {
		t.Fatalf("prefix does not render the image after the attachment:\n%s", prefix)
	}
}
