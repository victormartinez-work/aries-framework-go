/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tinkcrypto

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/golang/protobuf/proto"
	hybrid "github.com/google/tink/go/hybrid/subtle"
	"github.com/google/tink/go/keyset"
	tinkpb "github.com/google/tink/go/proto/tink_go_proto"

	ecdhespb "github.com/hyperledger/aries-framework-go/pkg/crypto/tinkcrypto/primitive/proto/ecdhes_aead_go_proto"
)

func extractPrivKey(kh *keyset.Handle) (*hybrid.ECPrivateKey, error) {
	buf := new(bytes.Buffer)
	w := &privKeyWriter{w: buf}
	nAEAD := &noopAEAD{}

	if kh == nil {
		return nil, fmt.Errorf("extractPrivKey: kh is nil")
	}

	err := kh.Write(w, nAEAD)
	if err != nil {
		return nil, fmt.Errorf("extractPrivKey: retrieving private key failed: %w", err)
	}

	ks := new(tinkpb.Keyset)

	err = proto.Unmarshal(buf.Bytes(), ks)
	if err != nil {
		return nil, errors.New("extractPrivKey: invalid private key")
	}

	ecdhesAESPrivateKeyTypeURL := "type.hyperledger.org/hyperledger.aries.crypto.tink.EcdhesAesAeadPrivateKey"
	primaryKey := ks.Key[0]

	if primaryKey.KeyData.TypeUrl != ecdhesAESPrivateKeyTypeURL {
		return nil, errors.New("extractPrivKey: can't extract unsupported private key")
	}

	pbKey := new(ecdhespb.EcdhesAeadPrivateKey)

	err = proto.Unmarshal(primaryKey.KeyData.Value, pbKey)
	if err != nil {
		return nil, errors.New("extractPrivKey: invalid key in keyset")
	}

	c, err := hybrid.GetCurve(pbKey.PublicKey.Params.KwParams.CurveType.String())
	if err != nil {
		return nil, fmt.Errorf("extractPrivKey: invalid key: %w", err)
	}

	return hybrid.GetECPrivateKey(c, pbKey.KeyValue), nil
}

type noopAEAD struct{}

func (n noopAEAD) Encrypt(plaintext, additionalData []byte) ([]byte, error) {
	return plaintext, nil
}

func (n noopAEAD) Decrypt(ciphertext, additionalData []byte) ([]byte, error) {
	return ciphertext, nil
}

type privKeyWriter struct {
	w io.Writer
}

// Write writes the public keyset to the underlying w.Writer. It's not used in this implementation.
func (p *privKeyWriter) Write(_ *tinkpb.Keyset) error {
	return fmt.Errorf("privKeyWriter: write function not supported")
}

// WriteEncrypted writes the encrypted keyset to the underlying w.Writer.
func (p *privKeyWriter) WriteEncrypted(ks *tinkpb.EncryptedKeyset) error {
	return write(p.w, ks)
}

func write(w io.Writer, ks *tinkpb.EncryptedKeyset) error {
	// we write EncryptedKeyset directly without decryption since noopAEAD was used to write *keyset.Handle
	_, e := w.Write(ks.EncryptedKeyset)
	return e
}
