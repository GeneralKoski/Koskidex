package eval

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// VarArchivio names the environment variable pointing at the thesis results
// folder. Every run is copied there as a new, timestamped file.
const VarArchivio = "TESI_RISULTATI"

// Archivia writes dati into $TESI_RISULTATI/sottocartella as a new file whose
// name starts with the current time. It never overwrites: the history of runs
// is what the charts are drawn from, and a run that replaces the previous one
// with the same name erases exactly the comparison the chart wants to show.
//
// With the variable unset it returns an empty path and no error, and the
// caller is expected to say so loudly: a result that was not archived must be
// visible as such, not discovered missing months later.
func Archivia(sottocartella, nome string, dati []byte) (string, error) {
	radice := os.Getenv(VarArchivio)
	if radice == "" {
		return "", nil
	}
	dir := filepath.Join(radice, sottocartella)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	base := time.Now().UTC().Format("2006-01-02T150405Z") + "_" + nome
	path := filepath.Join(dir, base+".json")
	for n := 2; ; n++ {
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err == nil {
			defer f.Close()
			_, err = f.Write(dati)
			return path, err
		}
		if !os.IsExist(err) {
			return "", err
		}
		path = filepath.Join(dir, fmt.Sprintf("%s-%d.json", base, n))
	}
}

// Provenienza says which code produced a run: the commit, and whether the tree
// had uncommitted changes. A number from a dirty tree is not reproducible from
// the commit alone, and the file has to say so.
func Provenienza(dir string) map[string]string {
	out := map[string]string{"commit": "sconosciuto", "modifiche_non_committate": "sconosciuto"}
	if b, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output(); err == nil {
		out["commit"] = strings.TrimSpace(string(b))
	}
	if b, err := exec.Command("git", "-C", dir, "status", "--porcelain").Output(); err == nil {
		out["modifiche_non_committate"] = fmt.Sprint(len(strings.TrimSpace(string(b))) > 0)
	}
	return out
}

// ImprontaFile is the SHA-256 of a file. Two runs claiming the same corpus
// can then be checked to have read the same bytes.
func ImprontaFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
