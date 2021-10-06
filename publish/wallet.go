package publish

import (
	"encoding/json"
	"io"
)

func LoadWallet(r io.Reader) (WalletFile, error) {
	var w WalletFile
	err := json.NewDecoder(r).Decode(&w)
	return w, err
}

type WalletFile struct {
	Name        string      `json:"name"`
	Version     int         `json:"version"`
	Preferences WalletPrefs `json:"preferences"`
	Accounts    []Account   `json:"accounts"`
}

type Account struct {
	AddressGenerator AddressGenerator  `json:"address_generator"`
	Certificates     map[string]string `json:"certificates"`
	Encrypted        bool              `json:"encrypted"`
	Ledger           string            `json:"ledger"`
	ModifiedOn       float64           `json:"modified_on"`
	Name             string            `json:"name"`
	PrivateKey       string            `json:"private_key"`
	PublicKey        string            `json:"public_key"`
	Seed             string            `json:"seed"`
}

type AddressGenerator struct {
	Name      string           `json:"name"`
	Change    AddressGenParams `json:"change"` // should "change" and "receiving" be replaced with a map[string]AddressGenParams?
	Receiving AddressGenParams `json:"receiving"`
}

type AddressGenParams struct {
	Gap                   int `json:"gap"`
	MaximumUsesPerAddress int `json:"maximum_uses_per_address"`
}

type WalletPrefs struct {
	Shared struct {
		Ts    float64 `json:"ts"`
		Value struct {
			Type  string `json:"type"`
			Value struct {
				AppWelcomeVersion int           `json:"app_welcome_version"`
				Blocked           []interface{} `json:"blocked"`
				Sharing3P         bool          `json:"sharing_3P"`
				Subscriptions     []string      `json:"subscriptions"`
				Tags              []string      `json:"tags"`
			} `json:"value"`
			Version string `json:"version"`
		} `json:"value"`
	} `json:"shared"`
}
