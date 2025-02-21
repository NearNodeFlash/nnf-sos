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
	"bytes"
	"testing"

	. "github.com/onsi/ginkgo/v2"
)

func TestToken(t *testing.T) {
	keyBytes1, pemKey1, err := CreateKeyForTokens()
	if err != nil {
		t.Errorf("failed to create key 1: %s", err.Error())
	}
	token1, err := CreateTokenFromKey(keyBytes1, "token-test")
	if err != nil {
		t.Errorf("failed to create token 1: %s", err.Error())
	}
	token2, err := CreateTokenFromKey(keyBytes1, "token-test")
	if err != nil {
		t.Errorf("failed to create token 2: %s", err.Error())
	}
	err = VerifyToken(token1, keyBytes1)
	if err != nil {
		t.Errorf("Validation of token1 failed: %s", err.Error())
	}
	err = VerifyToken(token2, keyBytes1)
	if err != nil {
		t.Errorf("Validation of token2 failed: %s", err.Error())
	}

	keyBytes1b, err := GetKeyFromPEM(pemKey1)
	if err != nil {
		t.Errorf("Validation of key and PEM failed: %s", err.Error())
	}
	if !bytes.Equal(keyBytes1, keyBytes1b) {
		t.Errorf("PEM encode/decode key mismatch")
	}

	keyBytes2, _, err := CreateKeyForTokens()
	if err != nil {
		t.Errorf("failed to create key 2: %s", err.Error())
	}
	err = VerifyToken(token1, keyBytes2)
	if err == nil {
		t.Error("Validation using wrong key succeeded but should have failed")
	}
}

// Just touch ginkgo, so it's here to interpret any ginkgo args from
// "make test", so that doesn't fail on this test file.
var _ = BeforeSuite(func() {})
