//
// Copyright 2023 IBM Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package resources

import (
	"crypto/rand"
	"math/big"
)

func RandString(length int) (string, error) {
	reader := rand.Reader
	outputStringByte := make([]byte, length)
	const randStringCharset string = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	randStringCharsetLength := big.NewInt(int64(len(randStringCharset)))
	for i := 0; i < length; i++ {
		charIndex, err := rand.Int(reader, randStringCharsetLength)
		if err != nil {
			return "", err
		}
		outputStringByte[i] = randStringCharset[charIndex.Int64()]
	}
	return string(outputStringByte), nil
}
