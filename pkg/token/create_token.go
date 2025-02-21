/*
 * Copyright 2025 Hewlett Packard Enterprise Development LP
 * Other additional copyright holders may be indicated within.
 *
 * The entirety of this work is licensed under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 *
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package token

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var signingMethod = jwt.SigningMethodHS256

// CreateTokenFromKey creates a new JSON Web Token (JWT) and signs it with the
// given key. The key should be a decoded PEM key, in DER form, such as the one
// produced by the CreateKeyForToken() function below.
func CreateTokenFromKey(key []byte, subject string) (string, error) {
	token := jwt.NewWithClaims(signingMethod,
		jwt.MapClaims{
			"sub": subject,
			"iat": time.Now().Unix(),
		})

	tokenString, err := token.SignedString(key)
	if err != nil {
		return "", fmt.Errorf("failure from SignedString: %w", err)
	}
	return tokenString, nil
}

// VerifyToken verifies that a given JWT was signed with the specified key.
func VerifyToken(tokenString string, key []byte) error {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != signingMethod.Name {
			return nil, errors.New("token verification failed: unexpected signing method for token")
		}
		return key, nil
	})
	if err != nil {
		return fmt.Errorf("token verification failed: parse failed: %w", err)
	}
	if !token.Valid {
		return errors.New("token verification failed: invalid token")
	}
	return nil
}

// CreateKeyForTokens creates an elliptic curve (EC) key that can be used for
// signing tokens. The token functions in this library will accept any valid
// key, EC or not, and do not require that the key be created by this function.
func CreateKeyForTokens() ([]byte, []byte, error) {
	// EC = elliptic curve. We're generating an ECDSA key.
	keyType := "EC PRIVATE KEY"
	privateKey, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	if err != nil {
		return []byte{}, []byte{}, fmt.Errorf("failure from GenerateKey: %w", err)
	}
	keyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return []byte{}, []byte{}, fmt.Errorf("failure from MarshalPKCS8PrivateKey: %w", err)
	}
	pemKey := pem.EncodeToMemory(&pem.Block{Type: keyType, Bytes: keyBytes})
	if pemKey == nil {
		return []byte{}, []byte{}, errors.New("unable to PEM-encode signing key for token")
	}
	return keyBytes, pemKey, nil
}

// GetKeyFromPEM will read the key from a PEM block.
func GetKeyFromPEM(pemKey []byte) ([]byte, error) {
	keyBlock, _ := pem.Decode(pemKey)
	if keyBlock == nil {
		return []byte{}, errors.New("unable to decode PEM block")
	}
	return keyBlock.Bytes, nil
}
