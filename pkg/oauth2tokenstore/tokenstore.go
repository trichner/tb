package oauth2tokenstore

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/trichner/tb/pkg/keyring"

	"golang.org/x/crypto/nacl/secretbox"

	"golang.org/x/oauth2"
)

func NewKeyringTokenStore(serviceName string) *KeyringTokenStore {
	return &KeyringTokenStore{
		serviceName:    serviceName,
		fileTokenStore: &secretBoxFileTokenStore{ringName: serviceName + "-keys"},
	}
}

type KeyringTokenStore struct {
	serviceName    string
	fileTokenStore *secretBoxFileTokenStore
}

func (k *KeyringTokenStore) Get(key string) (*oauth2.Token, error) {
	ring, err := keyring.Open(k.serviceName)
	if err != nil {
		return nil, err
	}

	item, err := ring.Get(key)
	if errors.Is(err, keyring.ErrNotFound) {
		return k.fileTokenStore.Get(key)
	} else if err != nil {
		return nil, fmt.Errorf("cannot read %s token for %q: %w", k.serviceName, key, err)
	}

	var token oauth2.Token
	err = json.Unmarshal([]byte(item.Secret), &token)
	if err != nil {
		return nil, fmt.Errorf("invalid %s token for %q: %w", k.serviceName, key, err)
	}

	return &token, nil
}

func (k *KeyringTokenStore) Put(key string, token *oauth2.Token) error {
	ring, err := keyring.Open(k.serviceName)
	if err != nil {
		return err
	}

	data, err := json.Marshal(token)
	if err != nil {
		return err
	}

	item := &keyring.Item{Secret: string(data)}
	err = ring.Put(key, item)
	if errors.Is(err, keyring.ErrTooBig) {
		log.Printf("Token for %q too large, was %d bytes. Storing token in encrypted file.", key, len(data))
		return k.fileTokenStore.Put(key, token)
	} else if err != nil {
		return fmt.Errorf("failed to store token for %s: %w", k.serviceName, err)
	}
	return nil
}

// secretBoxFileTokenStore implements a token store that writes tokens to encrypted files while
// storing the key to them in the keyring. This is useful if the tokens are too large.
type secretBoxFileTokenStore struct {
	ringName string
}

func (k *secretBoxFileTokenStore) Get(key string) (*oauth2.Token, error) {
	ring, err := keyring.Open(k.ringName)
	if err != nil {
		return nil, err
	}

	item, err := ring.Get(key)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil, nil
	} else if err != nil {
		k.cleanupKey(key)
		return nil, fmt.Errorf("cannot read %s token for %q: %w", k.ringName, key, err)
	}

	var secretKey [32]byte
	_, err = hex.Decode(secretKey[:], []byte(item.Secret))
	if err != nil {
		return nil, fmt.Errorf("failed to decode key for %q: %w", k.ringName, err)
	}

	f := deriveFileName(k.ringName, key)
	encrypted, err := os.ReadFile(f)
	if err != nil {
		k.cleanupKey(key)
		return nil, fmt.Errorf("failed to read token file %q: %w", f, err)
	}

	// When you decrypt, you must use the same nonce and key you used to
	// encrypt the message. One way to achieve this is to store the nonce
	// alongside the encrypted message. Above, we stored the nonce in the first
	// 24 bytes of the encrypted text.
	var decryptNonce [24]byte
	copy(decryptNonce[:], encrypted[:24])
	decrypted, ok := secretbox.Open(nil, encrypted[24:], &decryptNonce, &secretKey)
	if !ok {
		k.cleanupKey(key)
		return nil, fmt.Errorf("cannot decrypt token file")
	}

	var token oauth2.Token
	err = json.Unmarshal(decrypted, &token)
	if err != nil {
		k.cleanupKey(key)
		return nil, fmt.Errorf("invalid %s token for %q: %w", k.ringName, key, err)
	}

	return &token, nil
}

func (k *secretBoxFileTokenStore) cleanupKey(key string) {
	log.Printf("cleaning up %q - %q", k.ringName, key)
	ring, err := keyring.Open(k.ringName)
	if err == nil {
		err = ring.Put(key, nil)
		if err != nil {
			log.Printf("no token passphrase %q to remove: %s", key, err)
		}
	}

	f := deriveFileName(k.ringName, key)
	err = os.Remove(deriveFileName(k.ringName, key))
	if err != nil {
		log.Printf("no token file %q to remove: %s", f, err)
	}
}

func (k *secretBoxFileTokenStore) Put(key string, token *oauth2.Token) error {
	ring, err := keyring.Open(k.ringName)
	if err != nil {
		return err
	}

	data, err := json.Marshal(token)
	if err != nil {
		return err
	}

	var secretKey [32]byte
	if _, err := io.ReadFull(rand.Reader, secretKey[:]); err != nil {
		return fmt.Errorf("cannot generate cryptographic key: %w", err)
	}

	// You must use a different nonce for each message you encrypt with the
	// same key. Since the nonce here is 192 bits long, a random value
	// provides a sufficiently small probability of repeats.
	var nonce [24]byte
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		return fmt.Errorf("cannot generate cryptographic nonce: %w", err)
	}

	encrypted := secretbox.Seal(nonce[:], data, &nonce, &secretKey)

	f := deriveFileName(k.ringName, key)
	err = os.WriteFile(f, encrypted, 0o600)
	if err != nil {
		return fmt.Errorf("failed to write token file %q: %w", f, err)
	}

	// store key
	item := &keyring.Item{Secret: hex.EncodeToString(secretKey[:])}
	err = ring.Put(key, item)
	if errors.Is(err, keyring.ErrTooBig) {
		log.Printf("Token for %q too large, was %d bytes. Won't store token.", key, len(data))
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to store token for %s: %w", k.ringName, err)
	}
	return nil
}

func deriveFileName(ring string, key string) string {
	hashed := sha256.Sum256([]byte(ring + "-" + key))
	s := hex.EncodeToString(hashed[:16])
	return fmt.Sprintf("token_%s.bin", s)
}
