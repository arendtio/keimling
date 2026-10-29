package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	vaultVersion     = 1
	vaultFileName    = "credentials.vault"
	vaultEnvPassword = "KEIMLING_VAULT_PASSWORD"
)

// vaultEnvelope is the on-disk encrypted blob (KeePass-like single-password file).
type vaultEnvelope struct {
	Version    int    `json:"version"`
	Salt       []byte `json:"salt"`
	Nonce      []byte `json:"nonce"`
	Ciphertext []byte `json:"ciphertext"`
}

type vaultPayload struct {
	Credentials []Credential `json:"credentials"`
}

func credentialsVaultPath(configPath string) string {
	dir := filepath.Dir(strings.TrimSpace(configPath))
	if dir == "" || dir == "." {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config", "keimling")
	}
	return filepath.Join(dir, vaultFileName)
}

func deriveVaultKey(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, 3, 64*1024, 4, 32)
}

func encryptVaultPayload(key []byte, payload vaultPayload) (vaultEnvelope, error) {
	plain, err := json.Marshal(payload)
	if err != nil {
		return vaultEnvelope{}, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return vaultEnvelope{}, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return vaultEnvelope{}, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return vaultEnvelope{}, err
	}
	ct := gcm.Seal(nil, nonce, plain, nil)
	return vaultEnvelope{Version: vaultVersion, Nonce: nonce, Ciphertext: ct}, nil
}

func decryptVaultPayload(key []byte, env vaultEnvelope) (vaultPayload, error) {
	var empty vaultPayload
	if env.Version != 0 && env.Version != vaultVersion {
		return empty, fmt.Errorf("unsupported vault version %d", env.Version)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return empty, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return empty, err
	}
	plain, err := gcm.Open(nil, env.Nonce, env.Ciphertext, nil)
	if err != nil {
		return empty, fmt.Errorf("wrong password or corrupt vault")
	}
	var payload vaultPayload
	if err := json.Unmarshal(plain, &payload); err != nil {
		return empty, err
	}
	if payload.Credentials == nil {
		payload.Credentials = []Credential{}
	}
	return payload, nil
}

func readVaultEnvelope(path string) (vaultEnvelope, error) {
	var env vaultEnvelope
	b, err := os.ReadFile(path)
	if err != nil {
		return env, err
	}
	if err := json.Unmarshal(b, &env); err != nil {
		return env, err
	}
	return env, nil
}

func writeVaultEnvelope(path string, env vaultEnvelope) error {
	b, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func vaultPasswordFromEnv() (string, error) {
	return envOrFile(vaultEnvPassword)
}

func newVaultSalt() ([]byte, error) {
	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}
	return salt, nil
}
