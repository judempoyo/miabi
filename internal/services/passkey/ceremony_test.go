// Copyright (c) 2026 Jonas Kaninda.
//
// This file is part of Miabi.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package passkey

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/protocol/webauthncbor"
	"github.com/miabi-io/miabi/internal/models"
	"github.com/miabi-io/miabi/internal/storage/repositories"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const testOrigin = "https://miabi.example.com"

// softKey is a software authenticator holding one P-256 passkey.
type softKey struct {
	priv   *ecdsa.PrivateKey
	id     []byte
	handle []byte
	count  uint32
	rpHash [32]byte
}

func enc(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func clientData(typ string, challenge []byte) []byte {
	b, _ := json.Marshal(map[string]string{"type": typ, "challenge": enc(challenge), "origin": testOrigin})
	return b
}

func (k *softKey) authData(flags byte, attested []byte) []byte {
	out := append([]byte{}, k.rpHash[:]...)
	out = append(out, flags)
	out = binary.BigEndian.AppendUint32(out, k.count)
	return append(out, attested...)
}

func (k *softKey) create(t *testing.T, challenge []byte) []byte {
	t.Helper()
	cose, err := webauthncbor.Marshal(map[int]any{1: 2, 3: -7, -1: 1,
		-2: k.priv.PublicKey.X.FillBytes(make([]byte, 32)), -3: k.priv.PublicKey.Y.FillBytes(make([]byte, 32))})
	if err != nil {
		t.Fatal(err)
	}
	attested := append(make([]byte, 16), byte(len(k.id)>>8), byte(len(k.id)))
	attested = append(append(attested, k.id...), cose...)
	obj, err := webauthncbor.Marshal(map[string]any{"fmt": "none", "attStmt": map[string]any{}, "authData": k.authData(0x45, attested)})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(map[string]any{"id": enc(k.id), "rawId": enc(k.id), "type": "public-key",
		"response": map[string]string{"clientDataJSON": enc(clientData("webauthn.create", challenge)), "attestationObject": enc(obj)}})
	return b
}

func (k *softKey) assert(t *testing.T, challenge []byte, handle []byte) []byte {
	t.Helper()
	k.count++
	cd := clientData("webauthn.get", challenge)
	ad := k.authData(0x05, nil)
	sum := sha256.Sum256(cd)
	digest := sha256.Sum256(append(append([]byte{}, ad...), sum[:]...))
	sig, err := ecdsa.SignASN1(rand.Reader, k.priv, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(map[string]any{"id": enc(k.id), "rawId": enc(k.id), "type": "public-key",
		"response": map[string]string{"clientDataJSON": enc(cd), "authenticatorData": enc(ad), "signature": enc(sig), "userHandle": enc(handle)}})
	return b
}

func newCeremonyService(t *testing.T) (*Service, *repositories.PasskeyRepository) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Passkey{}); err != nil {
		t.Fatal(err)
	}
	repo := repositories.NewPasskeyRepository(db)
	cfg, _ := ConfigFromWebURL(testOrigin, "", "Miabi", nil)
	s, err := New(cfg, redis.NewClient(&redis.Options{Addr: miniredis.RunT(t).Addr()}), repo)
	if err != nil {
		t.Fatal(err)
	}
	return s, repo
}

func newSoftKey(t *testing.T) *softKey {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	id := make([]byte, 32)
	_, _ = rand.Read(id)
	return &softKey{priv: priv, id: id, rpHash: sha256.Sum256([]byte("miabi.example.com"))}
}

func register(t *testing.T, s *Service, user *models.User, k *softKey) *models.Passkey {
	t.Helper()
	ctx := context.Background()
	token, opts, err := s.BeginRegistration(ctx, user)
	if err != nil {
		t.Fatal(err)
	}
	k.handle = opts.Response.User.ID.(protocol.URLEncodedBase64)
	p, err := s.FinishRegistration(ctx, user, token, "Laptop", k.create(t, opts.Response.Challenge))
	if err != nil {
		t.Fatalf("FinishRegistration: %v", err)
	}
	return p
}

func login(s *Service, t *testing.T, k *softKey, handle []byte) (uint, error) {
	ctx := context.Background()
	token, opts, err := s.BeginLogin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return s.FinishLogin(ctx, token, k.assert(t, opts.Response.Challenge, handle))
}

func TestRegisterThenSignIn(t *testing.T) {
	s, repo := newCeremonyService(t)
	alice := &models.User{ID: 7, Email: "alice@example.com", Name: "Alice"}
	k := newSoftKey(t)
	p := register(t, s, alice, k)
	if p.UserID != 7 || p.Name != "Laptop" {
		t.Fatalf("stored passkey = %+v", p)
	}

	uid, err := login(s, t, k, k.handle)
	if err != nil || uid != 7 {
		t.Fatalf("FinishLogin = %d, %v", uid, err)
	}
	rows, _ := repo.ListByUser(7)
	if len(rows) != 1 || rows[0].LastUsedAt == nil {
		t.Fatalf("sign-in not recorded: %+v", rows)
	}

	// A second passkey reuses the account's handle, so either one resolves to the same user.
	k2 := newSoftKey(t)
	register(t, s, alice, k2)
	if string(k2.handle) != string(k.handle) {
		t.Fatal("a user's passkeys must share one user handle")
	}
}

func TestSignInRejectsForgedHandle(t *testing.T) {
	s, _ := newCeremonyService(t)
	k := newSoftKey(t)
	register(t, s, &models.User{ID: 7, Email: "alice@example.com"}, k)
	register(t, s, &models.User{ID: 8, Email: "bob@example.com"}, newSoftKey(t))

	// Alice's credential presented under someone else's handle must not sign anyone in.
	if _, err := login(s, t, k, []byte("not-alices-handle")); !errors.Is(err, ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
}

func TestSignInRejectsClonedKey(t *testing.T) {
	s, _ := newCeremonyService(t)
	k := newSoftKey(t)
	register(t, s, &models.User{ID: 7, Email: "alice@example.com"}, k)
	k.count = 10
	if _, err := login(s, t, k, k.handle); err != nil {
		t.Fatal(err)
	}
	k.count = 3
	if _, err := login(s, t, k, k.handle); !errors.Is(err, ErrInvalid) {
		t.Fatalf("a sign count that went backwards must be refused, got %v", err)
	}
}

func TestUnknownCredentialIsRefused(t *testing.T) {
	s, _ := newCeremonyService(t)
	stranger := newSoftKey(t)
	if _, err := login(s, t, stranger, []byte("whoever")); !errors.Is(err, ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
}
