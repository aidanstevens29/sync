package sync

import (
	"github.com/bugsnag/bugsnag-go"
	"github.com/olebedev/config"
	"io/ioutil"
	"sync/internal/pkg/sync"
)

// cfg contains the application's config
var cfg *config.Config

// Sync begins the sync process from HostBill to CRM
func Sync() {
	// Import our configuration data
	readConfig()

	// Setup our error notification platform
	sync.ConfigureBugsnag(cfg)

	// Sync accounts and get a map of HostBill IDs to Zoho IDs
	m := accounts(cfg)

	// Sync services and invoices
	go services(m, cfg)
	invoices(m, cfg)
}

// readConfig reads the config file and instantiates the config object
func readConfig() {
	file, err := ioutil.ReadFile("configs/config.yml")
	sync.PanicError(err)
	yamlString := string(file)
	cfg, err = config.ParseYaml(yamlString)
	sync.PanicError(err)
}

// invoices syncs all invoices from HostBill to CRM
func invoices(m map[string]string, cfg *config.Config) {
	totalPages := 0
	for {
		invoicesList, err := sync.DecodeInvoices(totalPages, cfg)
		if err != nil {
			_ = bugsnag.Notify(err)
			totalPages++
			continue
		}
		zohoInvoices, err := sync.ConvertInvoices(invoicesList, m)
		if err != nil {
			_ = bugsnag.Notify(err)
			totalPages++
			continue
		}
		err = sync.EncodeInvoices(zohoInvoices, cfg)
		if err != nil {
			_ = bugsnag.Notify(err)
			totalPages++
			continue
		}
		totalPages++
		if totalPages == invoicesList.Sorter.Totalpages {
			break
		}
	}
}

// services syncs all services from HostBill to CRM
func services(m map[string]string, cfg *config.Config) {
	totalPages := 0
	for {
		accountsList, err := sync.DecodeServices(totalPages, cfg)
		if err != nil {
			_ = bugsnag.Notify(err)
			totalPages++
			continue
		}
		zohoServices := sync.ConvertServices(accountsList, m)
		err = sync.EncodeServices(zohoServices, cfg)
		if err != nil {
			_ = bugsnag.Notify(err)
			totalPages++
			continue
		}
		totalPages++
		if totalPages == accountsList.Sorter.Totalpages {
			break
		}
	}
}

// accounts syncs all accounts from HostBill to CRM
func accounts(cfg *config.Config) map[string]string {
	totalPages := 0
	m := make(map[string]string)
	for {
		clientsList, err := sync.DecodeClients(totalPages, cfg)
		if err != nil {
			_ = bugsnag.Notify(err)
			totalPages++
			continue
		}
		zohoAccounts, err := sync.ConvertClients(clientsList, cfg)
		if err != nil {
			_ = bugsnag.Notify(err)
			totalPages++
			continue
		}
		body, err := sync.EncodeClients(zohoAccounts, cfg)
		if err != nil {
			_ = bugsnag.Notify(err)
			totalPages++
			continue
		}
		sync.CreateIdMap(body, m, clientsList)
		totalPages++
		if totalPages == clientsList.Sorter.Totalpages {
			break
		}
	}
	return m
}
