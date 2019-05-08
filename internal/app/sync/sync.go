package sync

import (
	"github.com/bugsnag/bugsnag-go"
	"sync/internal/pkg/config"
	"sync/internal/pkg/sync"
)

// Sync begins the sync process from HostBill to CRM
func Sync() {
	// Import our configuration data
	config.ReadConfig()

	// Setup our error notification platform
	config.ConfigureBugsnag()

	// Sync accounts and get a map of HostBill IDs to Zoho IDs
	m := accounts()

	// Sync services and invoices
	go services(m)
	invoices(m)
}

// invoices syncs all invoices from HostBill to CRM
func invoices(m map[string]string) {
	totalPages := 0
	for {
		invoicesList, err := sync.DecodeInvoices(totalPages)
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
		err = sync.EncodeInvoices(zohoInvoices)
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
func services(m map[string]string) {
	totalPages := 0
	for {
		accountsList, err := sync.DecodeServices(totalPages)
		if err != nil {
			_ = bugsnag.Notify(err)
			totalPages++
			continue
		}
		zohoServices := sync.ConvertServices(accountsList, m)
		err = sync.EncodeServices(zohoServices)
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
func accounts() map[string]string {
	totalPages := 0
	m := make(map[string]string)
	for {
		clientsList, err := sync.DecodeClients(totalPages)
		if err != nil {
			_ = bugsnag.Notify(err)
			totalPages++
			continue
		}
		zohoAccounts, err := sync.ConvertClients(clientsList)
		if err != nil {
			_ = bugsnag.Notify(err)
			totalPages++
			continue
		}
		body, err := sync.EncodeClients(zohoAccounts)
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
