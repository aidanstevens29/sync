package hostbill

import (
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"sync/internal/pkg/config"
	"sync/internal/pkg/request"
)

// Request makes a call to the HostBill API
func Request(call string, page int, id string) ([]byte, error) {
	domain, apiId, apiKey := config.SetHostBillConfig()
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
	res, err := request.Client.Do(req)
	if err != nil {
		return []byte(""), errors.Wrapf(err, "Failed to send query to HostBill %+v", req)
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			return
		}
	}()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return []byte(""), errors.Wrapf(err, "Failed to read response from HostBill %+v", req)
	}
	return body, nil
}
