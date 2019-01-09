package travis

/*
Copyright 2017 Shapath Neupane (@theshapguy)
Redistribution and use in source and binary forms, with or without modification, are permitted provided that the following conditions are met:
1. Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.
2. Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.
3. Neither the name of the copyright holder nor the names of its contributors may be used to endorse or promote products derived from this software without specific prior written permission.
THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
------------------------------------------------
Listener - written in Go because it's native web server is much more robust than Python. Plus its fun to write Go!
NOTE: Make sure you are using the right domain for travis [.com] or [.org]
Modified by wilsonk@lbry.io for LBRY internal-apis
*/

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"net/http"

	"github.com/lbryio/lbry.go/extras/errors"
)

func publicKey(isPrivateRepo bool) (*rsa.PublicKey, error) {
	var response *http.Response
	var err error
	if !isPrivateRepo {
		response, err = http.Get("https://api.travis-ci.org/config")
	} else {
		response, err = http.Get("https://api.travis-ci.com/config")
	}
	if err != nil {
		return nil, errors.Err("cannot fetch travis public key")
	}
	defer response.Body.Close()

	type configKey struct {
		Config struct {
			Notifications struct {
				Webhook struct {
					PublicKey string `json:"public_key"`
				} `json:"webhook"`
			} `json:"notifications"`
		} `json:"config"`
	}

	var t configKey

	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&t)
	if err != nil {
		return nil, errors.Err("cannot decode travis public key")
	}

	keyBlock, _ := pem.Decode([]byte(t.Config.Notifications.Webhook.PublicKey))
	if keyBlock == nil || keyBlock.Type != "PUBLIC KEY" {
		return nil, errors.Err("invalid travis public key")
	}

	publicKey, err := x509.ParsePKIXPublicKey(keyBlock.Bytes)
	if err != nil {
		return nil, errors.Err("invalid travis public key")
	}

	return publicKey.(*rsa.PublicKey), nil
}

func payloadDigest(payload string) []byte {
	hash := sha1.New()
	hash.Write([]byte(payload))
	return hash.Sum(nil)
}

func ValidateSignature(isPrivateRepo bool, r *http.Request) error {
	key, err := publicKey(isPrivateRepo)
	if err != nil {
		return errors.Err(err)
	}

	signature, err := base64.StdEncoding.DecodeString(r.Header.Get("Signature"))
	if err != nil {
		return errors.Err("cannot decode signature")
	}

	payload := payloadDigest(r.FormValue("payload"))

	err = rsa.VerifyPKCS1v15(key, crypto.SHA1, payload, signature)
	if err != nil {
		if err == rsa.ErrVerification {
			return errors.Err("invalid payload signature")
		}
		return errors.Err(err)
	}

	return nil
}

func NewFromRequest(r *http.Request) (*Webhook, error) {
	w := new(Webhook)

	err := json.Unmarshal([]byte(r.FormValue("payload")), w)
	if err != nil {
		return nil, errors.Err(err)
	}

	return w, nil
}
