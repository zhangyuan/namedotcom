package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/go-resty/resty/v2"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type DNSRecord struct {
	Id         int64  `json:"id"`
	DomainName string `json:"domainName"`
	Type       string `json:"type"`
	Answer     string `json:"answer"`
}

type DNSRecords = []DNSRecord

type DNSRecordsResponse struct {
	Records DNSRecords `json:"records"`
}

type CreateDNSRecordRequest struct {
	Host   string `json:"host"`
	Type   string `json:"type"`
	Answer string `json:"answer"`
	TTL    int    `json:"ttl"`
}

type UpdateDNSRecordRequest struct {
	Host   string `json:"host"`
	Type   string `json:"type"`
	Answer string `json:"answer"`
	TTL    int    `json:"ttl"`
}

func main() {
	if err := invoke(os.Args); err != nil {
		log.Fatalf("Error occurs %v.", err)
	}
}

func GetDNSRecords(client *resty.Request, domain string) (DNSRecords, error) {
	uri := fmt.Sprintf("https://api.name.com/v4/domains/%s/records", domain)

	resp, err := client.Get(uri)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("http response status code = %d", resp.StatusCode())
	}

	dnsRecordsResponse := DNSRecordsResponse{}

	responseBody := resp.Body()

	json.Unmarshal(responseBody, &dnsRecordsResponse)

	return dnsRecordsResponse.Records, nil
}

func verifyHttpResponse(resp *resty.Response) bool {
	return resp.StatusCode() >= 200 && resp.StatusCode() < 399
}

func CreateARecord(client *resty.Request, fqdn, value string) error {
	uri := fmt.Sprintf("https://api.name.com/v4/domains/%s/records", fqdn)

	host := GetHostFromFqdn(fqdn)

	requestBody := CreateDNSRecordRequest{
		Host:   host,
		Type:   "A",
		Answer: value,
		TTL:    300,
	}
	resp, err := client.SetBody(requestBody).Post(uri)

	if err != nil {
		return err
	}

	if !verifyHttpResponse(resp) {
		return fmt.Errorf("http status code %d and body %v", resp.StatusCode(), resp.String())
	}
	return nil
}

func UpdateARecord(client *resty.Request, fqdn string, recordId int64, value string) error {
	uri := fmt.Sprintf("https://api.name.com/v4/domains/%s/records/%d", fqdn, recordId)

	host := GetHostFromFqdn(fqdn)

	requestBody := UpdateDNSRecordRequest{
		Host:   host,
		Type:   "A",
		Answer: value,
		TTL:    300,
	}

	resp, err := client.SetBody(requestBody).Put(uri)

	if err != nil {
		return err
	}

	if !verifyHttpResponse(resp) {
		return fmt.Errorf("http status code %d and body %v", resp.StatusCode(), resp.String())
	}
	return nil
}

func GetDNSRecordId(request *resty.Request, domainName string) (int64, error) {
	dnsRecords, err := GetDNSRecords(request, domainName)
	if err != nil {
		return 0, err
	}

	for _, record := range dnsRecords {
		if record.Type == "A" {
			return record.Id, nil
		}
	}

	return 0, nil
}

func NewApiRequest(client *resty.Client) *resty.Request {
	username := os.Getenv("API_USERNAME")
	password := os.Getenv("API_PASSWORD")

	return client.R().
		EnableTrace().
		SetBasicAuth(username, password)
}

func GetHostFromFqdn(host string) string {
	parts := strings.Split(host, ".")
	number := 2

	return strings.Join(parts[0:len(parts)-number], ".")
}

func invoke(args []string) error {
	client := resty.New()

	fqdn := args[1]
	aRecordValue := args[2]

	dnsRecordId, err := GetDNSRecordId(NewApiRequest(client), fqdn)

	if err != nil {
		return err
	}

	if dnsRecordId == 0 {
		if err := CreateARecord(NewApiRequest(client), fqdn, aRecordValue); err != nil {
			return err
		}
	} else {
		if err := UpdateARecord(NewApiRequest(client), fqdn, dnsRecordId, aRecordValue); err != nil {
			return err
		}
	}

	newDnsRecords, err := GetDNSRecords(NewApiRequest(client), fqdn)
	if err != nil {
		return err
	}

	fmt.Println(newDnsRecords)

	return nil
}
