package hostbill

import (
	"github.com/olebedev/config"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
)

// client is the HTTP client to be used for all HostBill requests
var client = http.Client{}

// Request makes a call to the HostBill API
func Request(call string, page int, id string, cfg *config.Config) ([]byte, error) {
	domain, err, apiId, apiKey := setHostBillConfig(cfg)
	req, err := http.NewRequest(http.MethodGet, "https://"+domain+"/admin/api.php", nil)
	if err != nil {
		return []byte(""), errors.Wrapf(err, "Failed to create HostBill request %v", domain)
	}
	req.Close = true
	q := url.Values{}
	q.Add("api_id", apiId)
	q.Add("api_key", apiKey)
	q.Add("call", call)
	q.Add("page", strconv.Itoa(page))
	if id != "0" {
		q.Add("id", id)
	}
	req.URL.RawQuery = q.Encode()
	res, err := client.Do(req)
	if err != nil {
		return []byte(""), errors.Wrapf(err, "Failed to send query to HostBill %+v", req)
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			panicError(err)
		}
	}()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return []byte(""), errors.Wrapf(err, "Failed to read response from HostBill %+v", req)
	}
	return body, nil
}

// setHostBillConfig reads the relevant HostBill configuration values necessary to make a request to the HostBill API
func setHostBillConfig(cfg *config.Config) (string, error, string, string) {
	domain, err := cfg.String("hostbill.domain")
	panicError(err)
	apiId, err := cfg.String("hostbill.credentials.api_id")
	panicError(err)
	apiKey, err := cfg.String("hostbill.credentials.api_key")
	panicError(err)
	return domain, err, apiId, apiKey
}

// PanicError throws a panic if a fatal error has occurred
func panicError(err error) {
	if err != nil {
		panic(err)
	}
}
