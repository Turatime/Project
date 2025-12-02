package embedder

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type Client struct{ endpoint, key, model string }

func New(endpoint, key, model string) *Client { return &Client{endpoint, key, model} }

func (c *Client) Embed(texts []string) ([][]float32, error) {
	body := map[string]any{"model": c.model, "input": texts}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", strings.TrimRight(c.endpoint, "/")+"/v1/embeddings", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+c.key)
	req.Header.Set("Content-Type", "application/json")
	httpc := &http.Client{Timeout: 20 * time.Second}
	resp, err := httpc.Do(req)
	if err != nil { return nil, err }
	defer resp.Body.Close()
	var out struct{ Data []struct{ Embedding []float32 `json:"embedding"` } `json:"data"` }
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil { return nil, err }
	res := make([][]float32, len(out.Data))
	for i := range out.Data { res[i] = out.Data[i].Embedding }
	return res, nil
}

func FloatsToBytes(v []float32) []byte {
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.LittleEndian, v)
	return buf.Bytes()
}

func BytesToFloats(b []byte) []float32 {
	n := len(b) / 4
	out := make([]float32, n)
	_ = binary.Read(bytes.NewReader(b), binary.LittleEndian, &out)
	return out
}
