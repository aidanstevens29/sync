package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/bugsnag/bugsnag-go"
	"github.com/olebedev/config"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type ClientsList struct {
	Clients []struct {
		ID string `json:"id"`
	} `json:"clients"`
	Sorter struct {
		Totalpages int `json:"totalpages"`
	} `json:"sorter"`
}

type ZohoAccounts struct {
	ZohoAccountData      []ZohoAccountData `json:"data"`
	DuplicateCheckFields []string          `json:"duplicate_check_fields"`
}

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

type ZohoServices struct {
	ZohoServiceData      []ZohoServiceData `json:"data"`
	DuplicateCheckFields []string          `json:"duplicate_check_fields"`
}

type ZohoServiceData struct {
	RelatedAccount  string `json:"Related_Account"`
	BillingCycle    string `json:"Billing_Cycle"`
	Domain          string `json:"Domain"`
	ID              string `json:"ID1"`
	RecurringAmount string `json:"Recurring_Amount"`
	ServiceName     string `json:"Name"`
	Status          string `json:"Status"`
}

type ZohoInvoices struct {
	ZohoInvoiceData      []ZohoInvoiceData `json:"data"`
	DuplicateCheckFields []string          `json:"duplicate_check_fields"`
}

type ZohoInvoiceData struct {
	AccountName  string `json:"Account_Name"`
	DueDate    string `json:"Due_Date"`
	InvoiceDate          string `json:"Invoice_Date"`
	Paid              string `json:"Paid"`
	PaymentMethod string `json:"Payment_Method"`
	SubAmount     string `json:"Sub_Amount"`
	Subject          string `json:"Subject"`
	Total          string `json:"Total"`
	Status          string `json:"Status"`
	ID 			string `json:"ID1"`
}

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

type RefreshToken struct {
	AccessToken string `json:"access_token"`
}

type ZohoResponse struct {
	Data []struct {
		Details struct {
			ID string `json:"id"`
		} `json:"details"`
	} `json:"data"`
}

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

type InvoicesList struct {
	Invoices []struct {
		ID          string `json:"id"`
		Date        string `json:"date"`
		Duedate     string `json:"duedate"`
		Datepaid    string `json:"datepaid"`
		Subtotal2   string `json:"subtotal2"`
		Total       string `json:"total"`
		Status      string `json:"status"`
		ClientID    string `json:"client_id"`
		Module      string `json:"module"`
	} `json:"invoices"`
	Sorter struct {
		Totalpages    int    `json:"totalpages"`
	} `json:"sorter"`
}

// Zoho OAuth token
var code string

// HTTP client for all requests
var client = http.Client{}

// App config
var cfg *config.Config

func main() {
	// Import our configuration data
	readConfig()

	// Setup our error notification platform
	configureBugsnag()

	// Sync accounts and get map of HostBill IDs to Zoho IDs
	m := syncAccounts()

	// Sync services and invoices
	syncServices(m)
	syncInvoices(m)
}

func readConfig() {
	file, err := ioutil.ReadFile("config.yml")
	if err != nil {
		panic(err)
	}
	yamlString := string(file)
	cfg, err = config.ParseYaml(yamlString)
	if err != nil {
		panic(err)
	}
}

func configureBugsnag() {
	apiKey, err := cfg.String("bugsnag.credentials.api_key")
	if err != nil {
		panic(err)
	}
	bugsnag.Configure(bugsnag.Configuration{
		APIKey: apiKey,
	})
}

func syncInvoices (m map[string]string) {
	totalPages := 0
	for {
		body := hostbillRequest("getInvoices", totalPages, "0")
		invoicesList := InvoicesList{}
		err := json.Unmarshal(body, &invoicesList)
		if err != nil {
			_ = bugsnag.Notify(err)
			fmt.Println("Failed to decode HostBill invoice data. JSON:\n" + string(body))
		}

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
						_ = bugsnag.Notify(err)
						fmt.Println("Failed to decode invoice time and date. Date:\n" + v.Datepaid)
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

		upsertInvoices, err := json.Marshal(zohoInvoices)
		if err != nil {
			_ = bugsnag.Notify(err)
			fmt.Println("Failed to encode HostBill invoice data. JSON:\n" + string(body))
		}
		body = zohoRequest(upsertInvoices, "Invoices")
		if !strings.Contains(string(body), "FAILURE") {
			fmt.Printf("\nSucesfully synced %d invoices!", len(zohoInvoices.ZohoInvoiceData))
		} else {
			fmt.Println("Failed to sync some invoices.")
		}
		totalPages++
		if totalPages == invoicesList.Sorter.Totalpages {
			break
		}
	}
}

func syncServices(m map[string]string) {
	totalPages := 0
	for {
		body := hostbillRequest("getAccounts", totalPages, "0")
		accountsList := AccountsList{}
		err := json.Unmarshal(body, &accountsList)
		if err != nil {
			_ = bugsnag.Notify(err)
			fmt.Println("Failed to decode HostBill service data. JSON:\n" + string(body))
		}

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

		upsertServices, err := json.Marshal(zohoServices)
		if err != nil {
			_ = bugsnag.Notify(err)
			fmt.Println("Failed to encode HostBill invoice data. JSON:\n" + string(body))
		}
		body = zohoRequest(upsertServices, "Services")
		if !strings.Contains(string(body), "FAILURE") {
			fmt.Printf("\nSucesfully synced %d services!", len(zohoServices.ZohoServiceData))
		} else {
			fmt.Println("Failed to sync some services.")
		}
		totalPages++
		if totalPages == accountsList.Sorter.Totalpages {
			break
		}
	}
}

func syncAccounts() map[string]string {
	totalPages := 0
	m := make(map[string]string)
	for {
		body := hostbillRequest("getClients", totalPages, "0")
		clientsList := ClientsList{}
		err := json.Unmarshal(body, &clientsList)
		if err != nil {
			_ = bugsnag.Notify(err)
			fmt.Println("Failed to decode HostBill clients data. JSON:\n" + string(body))
		}

		zohoAccounts := ZohoAccounts{
			DuplicateCheckFields: []string{"Account_Number"},
		}

		for _, v := range clientsList.Clients {
			body := hostbillRequest("getClientDetails", 0, v.ID)
			clientDetails := ClientDetails{}
			err = json.Unmarshal(body, &clientDetails)
			if err != nil {
				_ = bugsnag.Notify(err)
				fmt.Println("Failed to decode HostBill client data. JSON:\n" + string(body))
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
		upsertAccounts, err := json.Marshal(zohoAccounts)
		if err != nil {
			_ = bugsnag.Notify(err)
			fmt.Println("Failed to encode HostBill client data. JSON:\n" + string(body))
		}
		refreshToken()
		body = zohoRequest(upsertAccounts, "Accounts")
		if !strings.Contains(string(body), "FAILURE") {
			fmt.Printf("\nSucesfully synced %d accounts!", len(zohoAccounts.ZohoAccountData))
		} else {
			fmt.Println("Failed to sync some accounts.")
		}
		createIdMap(body, m, clientsList)

		totalPages++
		if totalPages == clientsList.Sorter.Totalpages {
			break
		}
	}
	return m
}

func hostbillRequest(call string, page int, id string) []byte {
	req, err := http.NewRequest(http.MethodGet, "https://accounts.cartika.com/admin/api.php", nil)
	if err != nil {
		panic(err)
	}
	q := url.Values{}
	apiId, err := cfg.String("hostbill.credentials.api_id")
	if err != nil {
		panic(err)
	}
	apiKey, err := cfg.String("hostbill.credentials.api_key")
	if err != nil {
		panic(err)
	}
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
		panic(err)
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	return body
}

func zohoRequest(json []byte, module string) []byte {
	for {
		req, err := http.NewRequest("POST", "https://www.zohoapis.com/crm/v2/" + module + "/upsert", bytes.NewBuffer(json))
		if err != nil {
			panic(err)
		}
		req.Header.Set("Authorization", "Zoho-oauthtoken "+code)
		resp, err := client.Do(req)
		if err != nil {
			panic(err)
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		
		if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
			return body
		} else if resp.StatusCode == 401 {
			refreshToken()
		} else {
			_ = bugsnag.Notify(err)
			return body
		}
	}
}

func createIdMap(body []byte, m map[string]string, clientsList ClientsList) {
	zohoResponse := ZohoResponse{}
	err := json.Unmarshal(body, &zohoResponse)
	if err != nil {
		panic(err)
	}
	i := 0
	for _, x := range zohoResponse.Data {
		m[clientsList.Clients[i].ID] = x.Details.ID
		i++
		if i == len(clientsList.Clients) {
			break
		}
	}
}

func refreshToken() {
	client := http.Client{}
	req, err := http.NewRequest(http.MethodPost, "https://accounts.zoho.com/oauth/v2/token", nil)
	if err != nil {
		panic(err)
	}
	q := url.Values{}
	refresh, err := cfg.String("zoho.credentials.refresh_token")
	if err != nil {
		panic(err)
	}
	clientId, err := cfg.String("zoho.credentials.client_id")
	if err != nil {
		panic(err)
	}
	clientSecret, err := cfg.String("zoho.credentials.client_secret")
	if err != nil {
		panic(err)
	}
	q.Add("refresh_token", refresh)
	q.Add("client_id", clientId)
	q.Add("client_secret", clientSecret)
	q.Add("grant_type", "refresh_token")

	req.URL.RawQuery = q.Encode()

	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	refreshToken := RefreshToken{}
	err = json.Unmarshal(body, &refreshToken)
	if err != nil {
		panic(err)
	}
	code = refreshToken.AccessToken
}