package controllerImp

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/labstack/echo/v4"

	"aoi/pkg/kb/service"
)

type KBCtrl struct{
	s   service.KBService
	allow map[string]bool
	maxPages int
	maxBytes int
}

type ingestReq struct {
    Title     string  `json:"title"`
    Tags      string  `json:"tags"`
    Text      string  `json:"text"`
    SourceURL *string `json:"source_url"`
}

func New(s service.KBService) *KBCtrl {
	allow := map[string]bool{}
	for _, h := range strings.Split(os.Getenv("KB_ALLOWED_DOMAINS"), ",") {
		h = strings.TrimSpace(h)
		if h != "" { allow[strings.ToLower(h)] = true }
	}
	mp := 1; if v := os.Getenv("KB_MAX_PAGES_PER_JOB"); v != "" { fmt.Sscanf(v, "%d", &mp) }
	mb := 1500000; if v := os.Getenv("KB_MAX_BYTES_PER_PAGE"); v != "" { fmt.Sscanf(v, "%d", &mb) }
	return &KBCtrl{s: s, allow: allow, maxPages: mp, maxBytes: mb}
}

func (h *KBCtrl) IngestText(c echo.Context) error {
    var req ingestReq
    if err := c.Bind(&req); err != nil {
        return c.JSON(http.StatusBadRequest, map[string]any{"error": "invalid json: " + err.Error()})
    }
    if strings.TrimSpace(req.Title) == "" {
        return c.JSON(http.StatusBadRequest, map[string]any{"error": "title is required"})
    }
    if strings.TrimSpace(req.Text) == "" {
        return c.JSON(http.StatusBadRequest, map[string]any{"error": "text is required"})
    }

    // --- convert *string -> string ---
    src := ""
    if req.SourceURL != nil {
        src = *req.SourceURL
    }

    // NOTE: UpsertDocument คืน 3 ค่า → รับให้ครบ หรือทิ้งด้วย "_"
    if doc, chunks, err := h.s.UpsertDocument( // <-- use h.s (interface), not h.svc
    strings.TrimSpace(req.Title),
    strings.TrimSpace(req.Tags),
    req.Text,
    src,
	); err != nil {
		return c.JSON(http.StatusUnprocessableEntity, map[string]any{"error": err.Error()})
	} else {
		return c.JSON(http.StatusCreated, map[string]any{"doc": doc, "chunks": chunks})
	}
}

func (h *KBCtrl) IngestURL(c echo.Context) error {
	var body struct{ URL, Tags, Title string }
	if err := c.Bind(&body); err != nil || body.URL == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error":"url required"})
	}
	u, err := url.Parse(body.URL); if err != nil { return c.JSON(http.StatusBadRequest, map[string]string{"error":"bad url"}) }
	host := strings.ToLower(u.Host)
	if !h.allow[host] { return c.JSON(http.StatusForbidden, map[string]string{"error":"domain not allowed"}) }

	txt, title, err := fetchMainText(body.URL, h.maxBytes)
	if err != nil { return c.JSON(http.StatusBadGateway, map[string]string{"error":err.Error()}) }
	if body.Title != "" { title = body.Title }

	doc, n, err := h.s.UpsertDocument(title, body.Tags, txt, body.URL)
	if err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error":err.Error()}) }
	return c.JSON(http.StatusCreated, map[string]any{"doc": doc, "chunks": n})
}

func (h *KBCtrl) Search(c echo.Context) error {
    q := strings.TrimSpace(c.QueryParam("q"))
    if q == "" { return c.JSON(http.StatusBadRequest, map[string]string{"error":"q required"}) }

    chunks, err := h.s.Search(q, 6)
    if err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }

    // collect doc IDs
    seen := map[uint]struct{}{}
    ids := make([]uint, 0, len(chunks))
    for _, ch := range chunks {
        if _, ok := seen[ch.DocID]; !ok {
            seen[ch.DocID] = struct{}{}
            ids = append(ids, ch.DocID)
        }
    }
    meta, _ := h.s.DocsMeta(ids) // ignore error for now

    // shape response: include doc title + url for each chunk
    type outChunk struct {
        ChunkID  uint   `json:"chunk_id"`
        DocID    uint   `json:"doc_id"`
        Ord      int    `json:"ord"`
        Text     string `json:"text"`
        DocTitle string `json:"doc_title,omitempty"`
        SourceURL string `json:"source_url,omitempty"`
    }
    out := make([]outChunk, 0, len(chunks))
    for _, ch := range chunks {
        oc := outChunk{
            ChunkID: ch.ChunkID, DocID: ch.DocID, Ord: ch.Ord, Text: ch.Text,
        }
        if d, ok := meta[ch.DocID]; ok {
            oc.DocTitle = d.Title
            oc.SourceURL = d.SourceURL
        }
        out = append(out, oc)
    }
    return c.JSON(http.StatusOK, out)
}

// --- helpers ---
func fetchMainText(u string, maxBytes int) (string, string, error) {
	client := &http.Client{ Timeout: 20 * time.Second }
	resp, err := client.Get(u); if err != nil { return "", "", err }
	defer resp.Body.Close()
	if resp.ContentLength > 0 && resp.ContentLength > int64(maxBytes) { return "", "", fmt.Errorf("page too large") }
	limited := io.LimitedReader{R: resp.Body, N: int64(maxBytes)}
	b, err := io.ReadAll(&limited); if err != nil { return "", "", err }
	ct := strings.ToLower(resp.Header.Get("Content-Type"))
	// only html/plain for now
	if !strings.Contains(ct, "text/html") && !strings.Contains(ct, "text/plain") {
		return "", "", fmt.Errorf("unsupported content-type: %s", ct)
	}
	if strings.Contains(ct, "text/plain") {
		return string(b), guessTitleFromText(string(b)), nil
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(b)); if err != nil { return "", "", err }
	title := strings.TrimSpace(doc.Find("title").First().Text())

	// extract main content (simple rules: article/main + headers + p + li)
	var parts []string
	sel := doc.Find("main, article")
	if sel.Length() == 0 { sel = doc.Selection } // fallback
	sel.Find("h1,h2,h3,p,li").Each(func(_ int, s *goquery.Selection) {
		t := strings.TrimSpace(s.Text())
		if len(t) > 0 { parts = append(parts, t) }
	})
	text := cleanWhitespace(strings.Join(parts, "\n"))
	return text, title, nil
}

var wsRX = regexp.MustCompile(`\s+\n`)
func cleanWhitespace(s string) string { s = strings.ReplaceAll(s, "\r", ""); s = wsRX.ReplaceAllString(s, "\n"); return s }
func guessTitleFromText(s string) string {
	line := strings.SplitN(strings.TrimSpace(s), "\n", 2)[0]
	if len(line) > 120 { line = line[:120] }
	return line
}
