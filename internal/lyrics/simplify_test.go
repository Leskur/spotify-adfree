package lyrics

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSimplifyJSONTraditional(t *testing.T) {
	in := []byte(`{"lyrics":{"language":"zh-Hant","lines":[{"words":"到底要笑得多虛偽才能夠融入這世界?"}]}}`)
	out, err := SimplifyJSON(in)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if strings.Contains(s, "虛偽") || strings.Contains(s, "能夠") {
		t.Fatalf("still traditional: %s", s)
	}
	if !strings.Contains(s, "虚伪") && !strings.Contains(s, "能够") {
		// OpenCC 可能用不同用字，至少 language 应改
		var v map[string]any
		_ = json.Unmarshal(out, &v)
	}
	if !strings.Contains(s, "zh-Hans") {
		t.Fatalf("language not updated: %s", s)
	}
	if !strings.Contains(s, "虚伪") {
		t.Fatalf("expected simplified text: %s", s)
	}
}

func TestIsColorLyricsPath(t *testing.T) {
	if !IsColorLyricsPath("/color-lyrics/v2/track/abc/image/x") {
		t.Fatal("expected match")
	}
	if IsColorLyricsPath("/ad-logic/prefetch") {
		t.Fatal("should not match")
	}
}
