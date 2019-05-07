package zoho

import (
	"bytes"
	"encoding/json"
	"github.com/olebedev/config"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"net/url"
)

// RefreshToken models a response from the Zoho OAuth API
type RefreshToken struct {
	AccessToken string `json:"access_token"`
}

// code is our Zoho OAuth access token
var code string

// client is the HTTP client to be used for all HostBill requests
var client = http.Client{}

// RefreshAccessToken refreshes our OAuth access token
func RefreshAccessToken(cfg *config.Config) error {
	req, err := http.NewRequest(http.MethodPost, "https://accounts.zoho.com/oauth/v2/token", nil)
	if err != nil {
		return errors.Wrapf(err, "Failed to create Zoho OAuth request %v", req)
	}
	q := url.Values{}
	refresh, clientId, clientSecret := setZohoConfig(err, cfg)
	q.Add("refresh_token", refresh)
	q.Add("client_id", clientId)
	q.Add("client_secret", clientSecret)
	q.Add("grant_type", "refresh_token")

	req.URL.RawQuery = q.Encode()

	res, err := client.Do(req)
	if err != nil {
		return errors.Wrapf(err, "Failed to send OAuth query to Zoho %+v", req)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return errors.Wrapf(err, "Failed to read response from Zoho OAuth %+v", req)
	}
	refreshToken := RefreshToken{}
	err = json.Unmarshal(body, &refreshToken)
	if err != nil {
		return errors.Wrapf(err, "Failed to parse response from Zoho OAuth %v", string(body))
	}
	code = refreshToken.AccessToken
	return nil
}

// setZohoConfig reads the relevant Zoho configuration values necessary to make a request to the Zoho API
func setZohoConfig(err error, cfg *config.Config) (string, string, string) {
	refresh, err := cfg.String("zoho.credentials.refresh_token")
	panicError(err)
	clientId, err := cfg.String("zoho.credentials.client_id")
	panicError(err)
	clientSecret, err := cfg.String("zoho.credentials.client_secret")
	panicError(err)
	return refresh, clientId, clientSecret
}

// Request makes a request to the Zoho API
func Request(json []byte, module string, cfg *config.Config) ([]byte, error) {
	for {
		req, err := http.NewRequest("POST", "https://www.zohoapis.com/crm/v2/"+module+"/upsert", bytes.NewBuffer(json))
		if err != nil {
			return []byte(""), errors.Wrapf(err, "Failed to create Zoho request %v", module)
		}
		req.Close = true
		req.Header.Set("Authorization", "Zoho-oauthtoken "+code)
		resp, err := client.Do(req)
		if err != nil {
			return []byte(""), errors.Wrapf(err, "Failed to send query to Zoho %+v", req)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				panicError(err)
			}
		}()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return []byte(""), errors.Wrapf(err, "Failed to read response from HostBill %+v", req)
		}
		if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
			return body, nil
		} else if resp.StatusCode == 401 {
			err = RefreshAccessToken(cfg)
			if err != nil {
				return nil, err
			}
		} else {
			err = errors.New("Received a bad response code from Zoho:" + string(body))
			return body, err
		}
	}
}

// panicError throws a panic if a fatal error has occurred
func panicError(err error) {
	if err != nil {
		panic(err)
	}
}
