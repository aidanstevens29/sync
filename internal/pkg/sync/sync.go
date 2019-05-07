package sync

import (
	"encoding/json"
	"fmt"
	"github.com/bugsnag/bugsnag-go"
	"github.com/olebedev/config"
	"github.com/pkg/errors"
	"strings"
	"sync/internal/pkg/hostbill"
	"sync/internal/pkg/zoho"
	"time"
)

// ClientsList represents a list of clients as returned by the HostBill API
type ClientsList struct {
	Clients []struct {
		ID string `json:"id"`
	} `json:"clients"`
	Sorter struct {
		Totalpages int `json:"totalpages"`
	} `json:"sorter"`
}

// ZohoAccounts represents a list of accounts to be uploaded to Zoho
type ZohoAccounts struct {
	ZohoAccountData      []ZohoAccountData `json:"data"`
	DuplicateCheckFields []string          `json:"duplicate_check_fields"`
}

// ZohoAccountData represents an individual account's info to be uploaded to Zoho
type ZohoAccountData struct {
	AccountName    string `json:"Account_Name"`
	AccountNumber  string `json:"Account_Number"`
	AccountStatus  string `json:"Account_Status"`
	BillingCity    string `json:"Billing_City"`
	BillingCode    string `json:"Billing_Code"`
	BillingCountry string `json:"Billing_Country"`
	BillingState   string `json:"Billing_State"`
	BillingStreet  string `json:"Billing_Street"`
	Phone          string `json:"Phone"`
	Email          string `json:"Email"`
}

// ZohoServices represents a list of services to be uploaded to Zohoo
type ZohoServices struct {
	ZohoServiceData      []ZohoServiceData `json:"data"`
	DuplicateCheckFields []string          `json:"duplicate_check_fields"`
}

// ZohoServiceData represents information about individual services to be uploaded to Zoho
type ZohoServiceData struct {
	RelatedAccount  string `json:"Related_Account"`
	BillingCycle    string `json:"Billing_Cycle"`
	Domain          string `json:"Domain"`
	ID              string `json:"ID1"`
	RecurringAmount string `json:"Recurring_Amount"`
	ServiceName     string `json:"Name"`
	Status          string `json:"Status"`
}

// ZohoInvoices represents a list of invoices to be uploaded to Zoho
type ZohoInvoices struct {
	ZohoInvoiceData      []ZohoInvoiceData `json:"data"`
	DuplicateCheckFields []string          `json:"duplicate_check_fields"`
}

// ZohoInvoiceData represents info about an individual invoice to be uploaded to Zoho
type ZohoInvoiceData struct {
	AccountName   string `json:"Account_Name"`
	DueDate       string `json:"Due_Date"`
	InvoiceDate   string `json:"Invoice_Date"`
	Paid          string `json:"Paid"`
	PaymentMethod string `json:"Payment_Method"`
	SubAmount     string `json:"Sub_Amount"`
	Subject       string `json:"Subject"`
	Total         string `json:"Total"`
	Status        string `json:"Status"`
	ID            string `json:"ID1"`
}

// ClientDetails represents individual client's info downloaded from HostBill
type ClientDetails struct {
	Client struct {
		ID          string `json:"id"`
		Email       string `json:"email"`
		Password    string `json:"password"`
		Status      string `json:"status"`
		Firstname   string `json:"firstname"`
		Lastname    string `json:"lastname"`
		Companyname string `json:"companyname"`
		Address1    string `json:"address1"`
		City        string `json:"city"`
		State       string `json:"state"`
		Postcode    string `json:"postcode"`
		Country     string `json:"country"`
		Phonenumber string `json:"phonenumber"`
	} `json:"client"`
}

// ZohoResponse models a generic Zoho API response
type ZohoResponse struct {
	Data []struct {
		Details struct {
			ID string `json:"id"`
		} `json:"details"`
	} `json:"data"`
}

// AccountsList represents a list of services downloaded from HostBill
type AccountsList struct {
	Accounts []struct {
		ID           string `json:"id"`
		Domain       string `json:"domain"`
		Billingcycle string `json:"billingcycle"`
		Status       string `json:"status"`
		Total        string `json:"total"`
		Name         string `json:"name"`
		ClientID     string `json:"client_id"`
	} `json:"accounts"`
	Call   string `json:"call"`
	Sorter struct {
		Perpage    int `json:"perpage"`
		Totalpages int `json:"totalpages"`
	} `json:"sorter"`
}

// InvoicesList represents a list of invoices downloaded from HostBill
type InvoicesList struct {
	Invoices []struct {
		ID        string `json:"id"`
		Date      string `json:"date"`
		Duedate   string `json:"duedate"`
		Datepaid  string `json:"datepaid"`
		Subtotal2 string `json:"subtotal2"`
		Total     string `json:"total"`
		Status    string `json:"status"`
		ClientID  string `json:"client_id"`
		Module    string `json:"module"`
	} `json:"invoices"`
	Sorter struct {
		Totalpages int `json:"totalpages"`
	} `json:"sorter"`
}

// PanicError throws a panic if a fatal error has occurred
func PanicError(err error) {
	if err != nil {
		panic(err)
	}
}

// ConfigureBugsnag sets up bugsnag for panic reporting
func ConfigureBugsnag(cfg *config.Config) {
	apiKey, err := cfg.String("bugsnag.credentials.api_key")
	PanicError(err)
	bugsnag.Configure(bugsnag.Configuration{
		APIKey: apiKey,
	})
}

// EncodeInvoices encodes invoices into the Zoho API JSON format
func EncodeInvoices(zohoInvoices ZohoInvoices, cfg *config.Config) error {
	upsertInvoices, err := json.Marshal(zohoInvoices)
	if err != nil {
		return errors.Wrapf(err, "Failed to encode invoices %+v", zohoInvoices)
	}
	body, err := zoho.Request(upsertInvoices, "Invoices", cfg)
	if err != nil {
		return err
	}
	if !strings.Contains(string(body), "FAILURE") {
		fmt.Printf("\nSucesfully synced %d invoices!", len(zohoInvoices.ZohoInvoiceData))
	} else {
		fmt.Println("Failed to sync some invoices.")
	}
	return nil
}

// ConvertInvoices converts invoice data from the HostBill to the Zoho format
func ConvertInvoices(invoicesList InvoicesList, m map[string]string) (ZohoInvoices, error) {
	zohoInvoices := ZohoInvoices{
		DuplicateCheckFields: []string{"ID1"},
	}
	for _, v := range invoicesList.Invoices {
		if _, ok := m[v.ClientID]; ok {
			if v.Datepaid == "0000-00-00 00:00:00" {
				v.Datepaid = ""
			} else {
				paid, err := time.Parse("2006-01-02 15:04:05", v.Datepaid)
				if err != nil {
					return zohoInvoices, errors.Wrapf(err, "Failed to format date and time %v", v.Datepaid)
				}
				v.Datepaid = paid.Format("2006-01-02")
			}
			zohoInvoices.ZohoInvoiceData = append(zohoInvoices.ZohoInvoiceData, ZohoInvoiceData{
				AccountName:   m[v.ClientID],
				DueDate:       v.Duedate,
				InvoiceDate:   v.Date,
				Paid:          v.Datepaid,
				PaymentMethod: v.Module,
				SubAmount:     v.Subtotal2,
				Status:        v.Status,
				Total:         v.Total,
				Subject:       "Invoice #" + v.ID,
				ID:            v.ID,
			})
		}
	}
	return zohoInvoices, nil
}

// DecodeInvoices decodes invoice data from HostBill JSON
func DecodeInvoices(totalPages int, cfg *config.Config) (InvoicesList, error) {
	invoicesList := InvoicesList{}
	body, err := hostbill.Request("getInvoices", totalPages, "0", cfg)
	if err != nil {
		return invoicesList, err
	}
	err = json.Unmarshal(body, &invoicesList)
	if err != nil {
		return invoicesList, errors.Wrapf(err, "Failed to decode invoices %v", string(body))
	}
	return invoicesList, nil
}

// EncodeServices encodes services into the Zoho API JSON format
func EncodeServices(zohoServices ZohoServices, cfg *config.Config) error {
	upsertServices, err := json.Marshal(zohoServices)
	if err != nil {
		return errors.Wrapf(err, "Failed to encode services %+v", zohoServices)
	}
	body, err := zoho.Request(upsertServices, "Services", cfg)
	if err != nil {
		return err
	}
	if !strings.Contains(string(body), "FAILURE") {
		fmt.Printf("\nSucesfully synced %d services!", len(zohoServices.ZohoServiceData))
	} else {
		fmt.Println("Failed to sync some services.")
	}
	return nil
}

// ConvertServices converts service data from the HostBill to the Zoho format
func ConvertServices(accountsList AccountsList, m map[string]string) ZohoServices {
	zohoServices := ZohoServices{
		DuplicateCheckFields: []string{"ID1"},
	}
	for _, v := range accountsList.Accounts {
		zohoServices.ZohoServiceData = append(zohoServices.ZohoServiceData, ZohoServiceData{
			RelatedAccount:  m[v.ClientID],
			BillingCycle:    v.Billingcycle,
			Domain:          v.Domain,
			ID:              v.ID,
			RecurringAmount: v.Total,
			ServiceName:     v.Name,
			Status:          v.Status,
		})
	}
	return zohoServices
}

// DecodeServices decodes service data from HostBill JSON
func DecodeServices(totalPages int, cfg *config.Config) (AccountsList, error) {
	accountsList := AccountsList{}
	body, err := hostbill.Request("getAccounts", totalPages, "0", cfg)
	if err != nil {
		return accountsList, err
	}
	err = json.Unmarshal(body, &accountsList)
	if err != nil {
		return accountsList, errors.Wrapf(err, "Failed to decode services %v", string(body))
	}
	return accountsList, nil
}

// EncodeClients encodes accounts into the Zoho API JSON format
func EncodeClients(zohoAccounts ZohoAccounts, cfg *config.Config) ([]byte, error) {
	upsertAccounts, err := json.Marshal(zohoAccounts)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to encode accounts %+v", zohoAccounts)
	}
	err = zoho.RefreshAccessToken(cfg)
	if err != nil {
		return nil, err
	}
	body, err := zoho.Request(upsertAccounts, "Accounts", cfg)
	if err != nil {
		return nil, err
	}
	if !strings.Contains(string(body), "FAILURE") {
		fmt.Printf("\nSucesfully synced %d accounts!", len(zohoAccounts.ZohoAccountData))
	} else {
		fmt.Println("Failed to sync some accounts.")
	}
	return body, nil
}

// ConvertClients converts client data from the HostBill to the Zoho account format
func ConvertClients(clientsList ClientsList, cfg *config.Config) (ZohoAccounts, error) {
	zohoAccounts := ZohoAccounts{
		DuplicateCheckFields: []string{"Account_Number"},
	}
	for _, v := range clientsList.Clients {
		body, err := hostbill.Request("getClientDetails", 0, v.ID, cfg)
		if err != nil {
			return zohoAccounts, err
		}
		clientDetails := ClientDetails{}
		err = json.Unmarshal(body, &clientDetails)
		if err != nil {
			return zohoAccounts, errors.Wrapf(err, "Failed to decode HostBill client data %v", string(body))
		}
		if len(clientDetails.Client.Companyname) < 1 {
			clientDetails.Client.Companyname = clientDetails.Client.Firstname + " " + clientDetails.Client.Lastname
		}
		zohoAccounts.ZohoAccountData = append(zohoAccounts.ZohoAccountData, ZohoAccountData{
			AccountName:    clientDetails.Client.Companyname,
			AccountNumber:  clientDetails.Client.ID,
			AccountStatus:  clientDetails.Client.Status,
			BillingCity:    clientDetails.Client.City,
			BillingCode:    clientDetails.Client.Postcode,
			BillingCountry: clientDetails.Client.Country,
			BillingState:   clientDetails.Client.State,
			BillingStreet:  clientDetails.Client.Address1,
			Phone:          clientDetails.Client.Phonenumber,
			Email:          clientDetails.Client.Email,
		})
	}
	return zohoAccounts, nil
}

// DecodeClients decodes client data from HostBill JSON
func DecodeClients(totalPages int, cfg *config.Config) (ClientsList, error) {
	clientsList := ClientsList{}
	body, err := hostbill.Request("getClients", totalPages, "0", cfg)
	if err != nil {
		return clientsList, err
	}
	err = json.Unmarshal(body, &clientsList)
	if err != nil {
		return clientsList, errors.Wrapf(err, "Failed to decode clients %v", string(body))
	}
	return clientsList, nil
}

// CreateIdMap creates a map of HostBill IDs to Zoho IDs for use in syncing invoices and services
func CreateIdMap(body []byte, m map[string]string, clientsList ClientsList) {
	zohoResponse := ZohoResponse{}
	err := json.Unmarshal(body, &zohoResponse)
	PanicError(err)
	i := 0
	for _, x := range zohoResponse.Data {
		m[clientsList.Clients[i].ID] = x.Details.ID
		i++
		if i == len(clientsList.Clients) {
			break
		}
	}
}
