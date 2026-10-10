package lyrics

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/longbridgeapp/opencc"
)

var (
	t2sOnce sync.Once
	t2s     *opencc.OpenCC
	t2sErr  error
)

func converter() (*opencc.OpenCC, error) {
	t2sOnce.Do(func() {
		t2s, t2sErr = opencc.New("t2s")
	})
	return t2s, t2sErr
}

func IsColorLyricsPath(path string) bool {
	return strings.Contains(path, "/color-lyrics/")
}

// SimplifyResponse 将 color-lyrics JSON 中的繁体转为简体（就地改 resp）。
func SimplifyResponse(resp *http.Response) error {
	if resp == nil || resp.Body == nil {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil
	}
	ct := resp.Header.Get("Content-Type")
	if ct != "" && !strings.Contains(ct, "json") && !strings.Contains(ct, "text") {
		return nil
	}

	body, err := readDecodedBody(resp)
	if err != nil {
		return err
	}
	out, err := SimplifyJSON(body)
	if err != nil {
		// 转换失败则回退原文，避免歌词全挂
		resp.Body = io.NopCloser(bytes.NewReader(body))
		resp.ContentLength = int64(len(body))
		resp.Header.Del("Content-Encoding")
		resp.Header.Del("Content-Length")
		resp.Header.Set("Content-Length", strconv.Itoa(len(body)))
		return nil
	}
	resp.Body = io.NopCloser(bytes.NewReader(out))
	resp.ContentLength = int64(len(out))
	resp.Header.Del("Content-Encoding")
	resp.Header.Set("Content-Length", strconv.Itoa(len(out)))
	return nil
}

func readDecodedBody(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	var r io.Reader = resp.Body
	switch strings.ToLower(resp.Header.Get("Content-Encoding")) {
	case "gzip":
		gr, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		defer gr.Close()
		r = gr
	case "br", "zstd", "deflate":
		// 少见编码：原样读可能是压缩数据，交给上层失败回退困难；直接读原始字节
	}
	return io.ReadAll(r)
}

func SimplifyJSON(data []byte) ([]byte, error) {
	cc, err := converter()
	if err != nil {
		return nil, err
	}
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	convertValue(v, cc)
	return json.Marshal(v)
}

func convertValue(v any, cc *opencc.OpenCC) {
	switch x := v.(type) {
	case map[string]any:
		for k, child := range x {
			switch k {
			case "words", "transliteratedWords", "providerDisplayName":
				if s, ok := child.(string); ok && s != "" {
					if out, err := cc.Convert(s); err == nil {
						x[k] = out
					}
				}
			case "language":
				if s, ok := child.(string); ok {
					x[k] = toHansLang(s)
				}
			default:
				convertValue(child, cc)
			}
		}
	case []any:
		for _, child := range x {
			convertValue(child, cc)
		}
	}
}

func toHansLang(s string) string {
	s = strings.TrimSpace(s)
	switch strings.ToLower(s) {
	case "zh-hant", "zh-tw", "zh-hk", "zh-mo":
		return "zh-Hans"
	default:
		return s
	}
}
