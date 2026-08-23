package proof

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/timfewi/grepquest/internal/game"
)

type Receipt struct {
	ReceiptVersion         int            `json:"receipt_version"`
	PlayerAddress          string         `json:"player_address"`
	SeasonID               string         `json:"season_id"`
	PackSHA256             string         `json:"pack_sha256"`
	RulesSHA256            string         `json:"rules_sha256"`
	ClientBuildSHA256      string         `json:"client_build_sha256"`
	ChallengeResultsSHA256 string         `json:"challenge_results_sha256"`
	Score                  int            `json:"score"`
	Mastery                map[string]int `json:"mastery"`
	ProofTier              string         `json:"proof_tier"`
	CreatedAt              string         `json:"created_at"`
	SigningPublicKey       string         `json:"signing_public_key"`
	ReceiptSHA256          string         `json:"receipt_sha256"`
	Signature              string         `json:"signature"`
}

type Metadata struct {
	SeasonID, PackSHA256, RulesSHA256, ClientBuildSHA256 string
	CreatedAt                                            time.Time
}

// Level returns the deterministic level represented by the receipt score.
func (r Receipt) Level() int { return 1 + r.Score/300 }

func NewReceipt(profile game.Profile, publicKey ed25519.PublicKey, metadata Metadata) (Receipt, error) {
	if metadata.SeasonID == "" || len(metadata.PackSHA256) != 64 || len(metadata.RulesSHA256) != 64 || len(metadata.ClientBuildSHA256) != 64 {
		return Receipt{}, errors.New("complete season and SHA-256 metadata are required")
	}
	solved := make([]string, 0, len(profile.Solved))
	for id, ok := range profile.Solved {
		if ok {
			solved = append(solved, id)
		}
	}
	sort.Strings(solved)
	raw, _ := json.Marshal(solved)
	resultHash := sha256.Sum256(raw)
	pubHash := sha256.Sum256(publicKey)
	r := Receipt{ReceiptVersion: 1, PlayerAddress: "local:" + hex.EncodeToString(pubHash[:20]), SeasonID: metadata.SeasonID, PackSHA256: metadata.PackSHA256, RulesSHA256: metadata.RulesSHA256, ClientBuildSHA256: metadata.ClientBuildSHA256, ChallengeResultsSHA256: hex.EncodeToString(resultHash[:]), Score: profile.XP, Mastery: cloneMastery(profile.Mastery), ProofTier: "self", CreatedAt: metadata.CreatedAt.UTC().Format(time.RFC3339), SigningPublicKey: base64.RawStdEncoding.EncodeToString(publicKey)}
	return r, nil
}

func (r *Receipt) Sign(privateKey ed25519.PrivateKey) error {
	publicKey, ok := privateKey.Public().(ed25519.PublicKey)
	if !ok {
		return errors.New("invalid Ed25519 key")
	}
	encoded := base64.RawStdEncoding.EncodeToString(publicKey)
	if r.SigningPublicKey != encoded {
		return errors.New("receipt public key does not match signer")
	}
	canonical, err := r.Canonical()
	if err != nil {
		return err
	}
	sum := sha256.Sum256(canonical)
	r.ReceiptSHA256 = hex.EncodeToString(sum[:])
	r.Signature = base64.RawStdEncoding.EncodeToString(ed25519.Sign(privateKey, sum[:]))
	return nil
}
func (r Receipt) Verify() error {
	canonical, err := r.Canonical()
	if err != nil {
		return err
	}
	sum := sha256.Sum256(canonical)
	if !stringsEqualFold(r.ReceiptSHA256, hex.EncodeToString(sum[:])) {
		return errors.New("receipt hash mismatch")
	}
	publicKey, err := base64.RawStdEncoding.DecodeString(r.SigningPublicKey)
	if err != nil || len(publicKey) != ed25519.PublicKeySize {
		return errors.New("invalid signing public key")
	}
	signature, err := base64.RawStdEncoding.DecodeString(r.Signature)
	if err != nil || !ed25519.Verify(ed25519.PublicKey(publicKey), sum[:], signature) {
		return errors.New("invalid receipt signature")
	}
	return nil
}

// Canonical returns the version-1 byte representation. Field and mastery-key
// ordering are explicit and receipt_sha256/signature are intentionally absent.
func (r Receipt) Canonical() ([]byte, error) {
	if r.ReceiptVersion != 1 {
		return nil, errors.New("unsupported receipt version")
	}
	var b bytes.Buffer
	b.WriteByte('{')
	fields := []struct {
		name  string
		value any
	}{{"challenge_results_sha256", r.ChallengeResultsSHA256}, {"client_build_sha256", r.ClientBuildSHA256}, {"created_at", r.CreatedAt}}
	for i, f := range fields {
		if i > 0 {
			b.WriteByte(',')
		}
		writeJSONField(&b, f.name, f.value)
	}
	b.WriteString(`,"mastery":{`)
	keys := make([]string, 0, len(r.Mastery))
	for key := range r.Mastery {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for i, key := range keys {
		if i > 0 {
			b.WriteByte(',')
		}
		keyJSON, _ := json.Marshal(key)
		valueJSON, _ := json.Marshal(r.Mastery[key])
		b.Write(keyJSON)
		b.WriteByte(':')
		b.Write(valueJSON)
	}
	b.WriteByte('}')
	for _, f := range []struct {
		name  string
		value any
	}{{"pack_sha256", r.PackSHA256}, {"player_address", r.PlayerAddress}, {"proof_tier", r.ProofTier}, {"receipt_version", r.ReceiptVersion}, {"rules_sha256", r.RulesSHA256}, {"score", r.Score}, {"season_id", r.SeasonID}, {"signing_public_key", r.SigningPublicKey}} {
		b.WriteByte(',')
		writeJSONField(&b, f.name, f.value)
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}
func writeJSONField(b *bytes.Buffer, name string, value any) {
	nameJSON, _ := json.Marshal(name)
	valueJSON, _ := json.Marshal(value)
	b.Write(nameJSON)
	b.WriteByte(':')
	b.Write(valueJSON)
}
func cloneMastery(source map[string]int) map[string]int {
	result := make(map[string]int, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
func stringsEqualFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		x, y := a[i], b[i]
		if x >= 'A' && x <= 'F' {
			x += 32
		}
		if y >= 'A' && y <= 'F' {
			y += 32
		}
		if x != y {
			return false
		}
	}
	return true
}

func LoadOrCreateKey(path string) (ed25519.PrivateKey, error) {
	if key, err := loadKey(path); err == nil {
		return key, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, os.ErrExist) {
		return loadKey(path)
	}
	if err != nil {
		return nil, err
	}
	if _, err = file.Write(privateKey); err != nil {
		_ = file.Close()
		return nil, err
	}
	if err = file.Close(); err != nil {
		return nil, err
	}
	return privateKey, nil
}
func loadKey(path string) (ed25519.PrivateKey, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("signing key %s has unsafe permissions %o", path, info.Mode().Perm())
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(raw) != ed25519.PrivateKeySize {
		return nil, errors.New("invalid signing key length")
	}
	return ed25519.PrivateKey(raw), nil
}

func SHA256File(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()
	hash := sha256.New()
	if _, err = io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
