package serviceImp

import (
	"math"
	"sort"
	"strings"

	"aoi/entities"
	"aoi/pkg/kb/repository"
	"aoi/pkg/kb/embedder"
)


type Svc struct{ r repository.KBRepository; emb *embedder.Client }

func New(r repository.KBRepository, e *embedder.Client) *Svc { return &Svc{r: r, emb: e} }

func chunkText(text string, maxRunes int) []string {
	if maxRunes <= 0 { maxRunes = 1000 }
	parts := []string{}
	cur := strings.Builder{}
	count := 0
	for _, r := range text {
		cur.WriteRune(r); count++
		if count >= maxRunes && r == '\n' { parts = append(parts, cur.String()); cur.Reset(); count = 0 }
	}
	if cur.Len() > 0 { parts = append(parts, cur.String()) }
	return parts
}

func (s *Svc) UpsertDocument(title, tags, text, sourceURL string) (*entities.KBDocument, int, error) {
    d := &entities.KBDocument{Title: title, Tags: tags, SourceURL: sourceURL}
    if err := s.r.CreateDoc(d); err != nil { return nil, 0, err }

    chs := chunkText(text, 1000)
    if len(chs) == 0 { return d, 0, nil }

    var embs [][]float32
    var err error
    if s.emb != nil {
        embs, err = s.emb.Embed(chs)
        if err != nil {
            // degrade gracefully: keep chunks with empty embeddings
            embs = nil
        }
    }

    rows := make([]entities.KBChunk, len(chs))
    for i := range chs {
        var embBytes []byte
        if embs != nil && i < len(embs) {
            embBytes = embedder.FloatsToBytes(embs[i])
        } // else keep nil []byte → safe, cosine() will treat as zero-vector
        rows[i] = entities.KBChunk{
            DocID:     d.DocID,
            Ord:       i,
            Text:      chs[i],
            Embedding: embBytes,
        }
    }

    if err := s.r.BulkInsertChunks(rows); err != nil { return nil, 0, err }
    return d, len(rows), nil
}

func cosine(a, b []float32) float64 {
	var dot, na, nb float64
	for i := 0; i < len(a) && i < len(b); i++ {
		dot += float64(a[i] * b[i]); na += float64(a[i]*a[i]); nb += float64(b[i]*b[i])
	}
	if na == 0 || nb == 0 { return 0 }
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

func (s *Svc) Search(query string, k int) ([]entities.KBChunk, error) {
	q := strings.TrimSpace(query)
	if q == "" || k <= 0 {
		return nil, nil
	}

	// 1) Try to embed the query (safe if emb is nil or embedding fails)
	var qvec []float32
	if s.emb != nil {
		if vec, err := s.emb.Embed([]string{q}); err == nil && len(vec) > 0 { // ← use your embedder's actual method name
			qvec = vec[0]
		}
	}

	// 2) Fetch candidate chunks from repo
	chunks, err := s.r.AllChunks()
	if err != nil {
		return nil, err
	}
	if len(chunks) == 0 {
		return nil, nil
	}

	// 3) Score candidates
	type scored struct {
		ch entities.KBChunk
		sc float64
	}
	scoredList := make([]scored, 0, len(chunks))

	if len(qvec) > 0 {
		// vector similarity
		for _, ch := range chunks {
			chVec := embedder.BytesToFloats(ch.Embedding)
			if len(chVec) == 0 || len(chVec) != len(qvec) {
				continue
			}
			var dot, nq, nd float64
			for i := range qvec {
				v := float64(qvec[i])
				w := float64(chVec[i])
				dot += v * w
				nq += v * v
				nd += w * w
			}
			if nq == 0 || nd == 0 {
				continue
			}
			sc := dot / (math.Sqrt(nq) * math.Sqrt(nd))
			scoredList = append(scoredList, scored{ch: ch, sc: sc})
		}
	} else {
		// keyword fallback
		qlow := strings.ToLower(q)
		for _, ch := range chunks {
			score := 0.0
			if strings.Contains(strings.ToLower(ch.Text), qlow) {
				score = 1.0
			}
			scoredList = append(scoredList, scored{ch: ch, sc: score})
		}
	}

	if len(scoredList) == 0 {
		return nil, nil
	}
	sort.Slice(scoredList, func(i, j int) bool { return scoredList[i].sc > scoredList[j].sc })

	if k > len(scoredList) {
		k = len(scoredList)
	}
	out := make([]entities.KBChunk, 0, k)
	for i := 0; i < k; i++ {
		out = append(out, scoredList[i].ch)
	}
	return out, nil
}


func (s *Svc) DocsMeta(ids []uint) (map[uint]entities.KBDocument, error) {
	return s.r.DocsByIDs(ids)
}