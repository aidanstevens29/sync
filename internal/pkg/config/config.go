package config

import (
	"github.com/bugsnag/bugsnag-go"
	"github.com/olebedev/config"
	"io/ioutil"
)

// Cfg contains the application's config
var Cfg *config.Config

// ReadConfig reads the config file and instantiates the config object
func ReadConfig() {
	file, err := ioutil.ReadFile("configs/config.yml")
	panicError(err)
	yamlString := string(file)
	Cfg, err = config.ParseYaml(yamlString)
	panicError(err)
}

// SetZohoConfig reads the relevant Zoho configuration values necessary to make a request to the Zoho API
func SetZohoConfig() (string, string, string) {
	refresh, err := Cfg.String("zoho.credentials.refresh_token")
	panicError(err)
	clientId, err := Cfg.String("zoho.credentials.client_id")
	panicError(err)
	clientSecret, err := Cfg.String("zoho.credentials.client_secret")
	panicError(err)
	return refresh, clientId, clientSecret
}

// SetHostBillConfig reads the relevant HostBill configuration values necessary to make a request to the HostBill API
func SetHostBillConfig() (string, string, string) {
	domain, err := Cfg.String("hostbill.domain")
	panicError(err)
	apiId, err := Cfg.String("hostbill.credentials.api_id")
	panicError(err)
	apiKey, err := Cfg.String("hostbill.credentials.api_key")
	panicError(err)
	return domain, apiId, apiKey
}

// ConfigureBugsnag sets up bugsnag for panic reporting
func ConfigureBugsnag() {
	apiKey, err := Cfg.String("bugsnag.credentials.api_key")
	panicError(err)
	bugsnag.Configure(bugsnag.Configuration{
		APIKey: apiKey,
	})
}

// panicError throws a panic if a fatal error has occurred
func panicError(err error) {
	if err != nil {
		panic(err)
	}
}
