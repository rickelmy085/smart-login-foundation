package knowledge

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)
// ScannedFile é um arquivo descoberto pelo scanner, já com hash.
// Usamos o hash como ID estável no banco: se o mesmo arquivo for
// re-ingerido, identificamos por hash e pulamos trabalho.
type ScannedFile struct {
	Path  string
	Title string
	Hash  string // SHA-256 hex
	Size  int64
}

// ScanDir varre `root` recursivamente e devolve apenas os PDFs
// (extensão .pdf, case-insensitive). Resultado vem ordenado por nome
// para reprodutibilidade.
//
// Idempotente e side-effect-free: só lê, não modifica nada em disco.
func ScanDir(root string) ([]ScannedFile, error) {
	// Normaliza separadores para o SO atual (importante em Windows
	// quando o caminho chega com "/" vindos de CLI/flag).
	root = filepath.FromSlash(root)

	var out []ScannedFile

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !strings.EqualFold(filepath.Ext(d.Name()), ".pdf") {
			return nil
		}
		sf, err := scanFile(path)
		if err != nil {
			// Não falhamos o batch por causa de um PDF ruim; só logamos
			// via erro retornado e seguimos.
			return fmt.Errorf("scan %s: %w", path, err)
		}
		out = append(out, sf)
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Title < out[j].Title })
	return out, nil
}

// scanFile lê o arquivo e calcula seu SHA-256. Title é o nome sem extensão.
func scanFile(path string) (ScannedFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return ScannedFile{}, err
	}
	defer f.Close()

	h := sha256.New()
	size, err := io.Copy(h, f)
	if err != nil {
		return ScannedFile{}, err
	}

	title := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return ScannedFile{
		Path:  path,
		Title: title,
		Hash:  hex.EncodeToString(h.Sum(nil)),
		Size:  size,
	}, nil
}
