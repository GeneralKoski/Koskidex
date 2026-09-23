package eval

import (
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

// GradoMax is the highest relevance grade a judgment may carry. Three levels,
// like NFCorpus: 0 not relevant, 1 relevant, 2 exactly what was being looked
// for.
const GradoMax = 2

// SheetRow is one line of the annotation sheet: a query, a document, and the
// grade to fill in.
//
// Snippet is there so the document can be judged without opening it. Title and
// Snippet are never read back: only QueryID, DocID and Grade end up in the
// qrels.
type SheetRow struct {
	QueryID string
	Query   string
	DocID   string
	Title   string
	Snippet string
	Grade   string
}

var intestazioneFoglio = []string{"query_id", "query", "doc_id", "titolo", "estratto", "grado"}

// PoolForQuery is the set of documents to judge for one query: the union of
// the top depth results of every run.
//
// The result is sorted by document ID, and that is the point rather than a
// detail. Presented in ranked order, or grouped by which run found what, the
// assessor reads the ranking as a hint and confirms it; the judgments then
// agree with the system that produced them and the evaluation measures
// nothing. TREC shuffles the pool for the same reason.
//
// The bias that remains cannot be sorted away: a document no run retrieved is
// never judged, and counts as not relevant. Recall measured against a pooled
// collection is therefore an upper bound, and it is only comparable between
// runs that contributed to the pool.
func PoolForQuery(rankings [][]string, depth int) []string {
	visti := map[string]bool{}
	for _, r := range rankings {
		for i, id := range r {
			if i >= depth {
				break
			}
			visti[id] = true
		}
	}

	out := make([]string, 0, len(visti))
	for id := range visti {
		out = append(out, id)
	}
	sort.Strings(out)

	return out
}

// WriteSheet writes the annotation sheet as a tab-separated file, which is
// what opens in a spreadsheet without an import dialog mangling the text.
func WriteSheet(w io.Writer, righe []SheetRow) error {
	out := csv.NewWriter(w)
	out.Comma = '\t'

	if err := out.Write(intestazioneFoglio); err != nil {
		return err
	}
	for _, r := range righe {
		if err := out.Write([]string{r.QueryID, r.Query, r.DocID, r.Title, pulisci(r.Snippet), r.Grade}); err != nil {
			return err
		}
	}
	out.Flush()

	return out.Error()
}

// ReadSheet reads back a sheet, filled in or not.
func ReadSheet(r io.Reader) ([]SheetRow, error) {
	in := csv.NewReader(r)
	in.Comma = '\t'
	in.FieldsPerRecord = len(intestazioneFoglio)

	record, err := in.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(record) == 0 {
		return nil, fmt.Errorf("foglio vuoto: manca perfino l'intestazione")
	}
	if record[0][0] != intestazioneFoglio[0] {
		return nil, fmt.Errorf("prima colonna %q, attesa %q: non sembra un foglio di annotazione", record[0][0], intestazioneFoglio[0])
	}

	righe := make([]SheetRow, 0, len(record)-1)
	for _, c := range record[1:] {
		righe = append(righe, SheetRow{
			QueryID: c[0], Query: c[1], DocID: c[2], Title: c[3], Snippet: c[4], Grade: c[5],
		})
	}

	return righe, nil
}

// SheetToQrels turns a filled sheet into relevance judgments.
//
// An unfilled grade is an error, not a zero. The two are not the same thing:
// "I looked at it and it is not relevant" is a judgment, "I never got to it"
// is a hole, and silently reading the second as the first inflates every
// number computed from the file.
func SheetToQrels(righe []SheetRow) (Qrels, error) {
	q := Qrels{}
	var mancanti []string

	for i, r := range righe {
		g := strings.TrimSpace(r.Grade)
		if g == "" {
			mancanti = append(mancanti, fmt.Sprintf("riga %d (%s / %s)", i+2, r.QueryID, r.DocID))

			continue
		}
		n, err := strconv.Atoi(g)
		if err != nil || n < 0 || n > GradoMax {
			return nil, fmt.Errorf("riga %d: grado %q non valido, atteso un intero fra 0 e %d", i+2, r.Grade, GradoMax)
		}
		if q[r.QueryID] == nil {
			q[r.QueryID] = map[string]int{}
		}
		q[r.QueryID][r.DocID] = n
	}

	if len(mancanti) > 0 {
		return nil, fmt.Errorf("%d giudizi non compilati, il primo e' %s: un grado vuoto non e' uno zero", len(mancanti), mancanti[0])
	}

	return q, nil
}

// WriteQrels writes judgments in BEIR's qrels format, so the collection is
// read by the same loader as SciFact and NFCorpus.
func WriteQrels(w io.Writer, q Qrels) error {
	if _, err := fmt.Fprintln(w, "query-id\tcorpus-id\tscore"); err != nil {
		return err
	}

	queries := make([]string, 0, len(q))
	for id := range q {
		queries = append(queries, id)
	}
	sort.Strings(queries)

	for _, qid := range queries {
		docs := make([]string, 0, len(q[qid]))
		for id := range q[qid] {
			docs = append(docs, id)
		}
		sort.Strings(docs)
		for _, did := range docs {
			if _, err := fmt.Fprintf(w, "%s\t%s\t%d\n", qid, did, q[qid][did]); err != nil {
				return err
			}
		}
	}

	return nil
}

// pulisci flattens a snippet onto one line: a newline or a tab inside a field
// would break the sheet open in a spreadsheet.
func pulisci(s string) string {
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")

	return strings.Join(strings.Fields(s), " ")
}
