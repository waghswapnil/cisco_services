package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

var SsoUrl = "https://cloudsso.cisco.com/as/token.oauth2"
var Sn2infoStatusUrl = "https://api.cisco.com/sn2info/v2/coverage/summary/serial_numbers/"
var ProductInfoUrl = "https://api.cisco.com/product/v1/information/serial_numbers/"
var debug = false

type OAuth struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type Sn2infoSummary struct {
	PaginationResponseRecord struct {
		LastIndex    int    `json:"last_index"`
		PageIndex    int    `json:"page_index"`
		PageRecords  int    `json:"page_records"`
		SelfLink     string `json:"self_link"`
		Title        string `json:"title"`
		TotalRecords int    `json:"total_records"`
	} `json:"pagination_response_record"`
	SerialNumbers []struct {
		BasePidList []struct {
			BasePid string `json:"base_pid"`
		} `json:"base_pid_list"`
		ContractSiteCustomerName  string `json:"contract_site_customer_name"`
		ContractSiteAddress1      string `json:"contract_site_address1"`
		ContractSiteCity          string `json:"contract_site_city"`
		ContractSiteStateProvince string `json:"contract_site_state_province"`
		ContractSiteCountry       string `json:"contract_site_country"`
		CoveredProductLineEndDate string `json:"covered_product_line_end_date"`
		ID                        string `json:"id"`
		IsCovered                 string `json:"is_covered"`
		OrderablePidList          []struct {
			ItemDescription string `json:"item_description"`
			ItemPosition    string `json:"item_position"`
			ItemType        string `json:"item_type"`
			OrderablePid    string `json:"orderable_pid"`
			PillarCode      string `json:"pillar_code"`
		} `json:"orderable_pid_list"`
		ParentSrNo              string `json:"parent_sr_no"`
		ServiceContractNumber   string `json:"service_contract_number"`
		ServiceLineDescr        string `json:"service_line_descr"`
		SrNo                    string `json:"sr_no"`
		WarrantyEndDate         string `json:"warranty_end_date"`
		WarrantyType            string `json:"warranty_type"`
		WarrantyTypeDescription string `json:"warranty_type_description"`
	} `json:"serial_numbers"`
}

type ProductInfoSummary struct {
	PaginationResponseRecord struct {
		LastIndex    int    `json:"last_index"`
		PageIndex    int    `json:"page_index"`
		PageRecords  int    `json:"page_records"`
		SelfLink     string `json:"self_link"`
		Title        string `json:"title"`
		TotalRecords int    `json:"total_records"`
	} `json:"pagination_response_record"`
	ProductList []struct {
		ID                 string `json:"id"`
		SrNo               string `json:"sr_no"`
		BasePid            string `json:"base_pid"`
		OrderablePid       string `json:"orderable_pid"`
		ProductName        string `json:"product_name"`
		ProductType        string `json:"product_type"`
		ProductSeries      string `json:"product_series"`
		ProductCategory    string `json:"product_category"`
		ProductSubcategory string `json:"product_subcategory"`
		ReleaseDate        string `json:"release_date"`
		OrderableStatus    string `json:"orderable_status"`
		Dimensions         struct {
			DimensionsFormat string `json:"dimensions_format"`
			DimensionsValue  string `json:"dimensions_value"`
		} `json:"dimensions"`
		Weight             string `json:"weight"`
		FormFactor         string `json:"form_factor"`
		ProductSupportPage string `json:"product_support_page"`
		VisioStencilURL    string `json:"visio_stencil_url"`
		RichMediaUrls      struct {
			SmallImageURL string `json:"small_image_url"`
			LargeImageURL string `json:"large_image_url"`
		} `json:"rich_media_urls"`
	} `json:"product_list"`
}

func auth(url, clientId, clientSecret, grantType string) (OAuth, error) {
	var token OAuth
	params := "client_id=" + clientId + "&" + "client_secret=" + clientSecret + "&" + "grant_type=" + grantType
	body := strings.NewReader(params)
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return token, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return token, err
	}
	defer resp.Body.Close()

	rspBody, _ := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(rspBody, &token)
	if err != nil {
		return token, err
	}
	return token, err
}

func prettyprint(b []byte) ([]byte, error) {
	var out bytes.Buffer
	err := json.Indent(&out, b, "", "  ")
	return out.Bytes(), err
}

func send_request(url, authToken string) (*http.Response, error) {
	var rsp *http.Response
	if debug {
		fmt.Println("====>" + url)
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return rsp, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+authToken)

	rsp, err = http.DefaultClient.Do(req)
	if err != nil {
		return rsp, err
	}
	return rsp, nil
}

func SN2INFO_request(url, serial, authToken string) (Sn2infoSummary, error) {
	var sn2info Sn2infoSummary
	rsp, err := send_request(url+serial, authToken)
	if err != nil {
		return sn2info, err
	}

	defer rsp.Body.Close()
	rspBody, _ := ioutil.ReadAll(rsp.Body)
	err = json.Unmarshal(rspBody, &sn2info)
	if err != nil {
		return sn2info, err
	}
	if debug {
		b, _ := prettyprint(rspBody)
		fmt.Printf("%s", b)
	}
	return sn2info, err
}

func ProductInfo_request(url, serial, authToken string) (ProductInfoSummary, error) {
	var info ProductInfoSummary
	rsp, err := send_request(url+serial, authToken)
	if err != nil {
		return info, err
	}

	defer rsp.Body.Close()
	rspBody, _ := ioutil.ReadAll(rsp.Body)
	err = json.Unmarshal(rspBody, &info)
	if err != nil {
		return info, err
	}
	if debug {
		b, _ := prettyprint(rspBody)
		fmt.Printf("%s", b)
	}
	return info, err
}

func main() {
	serialPtr := flag.String("serial", "", "Serial Number")
	debugPtr := flag.Bool("debug", false, "Debug Mode")
	flag.Parse()

	authToken := os.Getenv("AUTH_TOKEN")
	if authToken == "" {
		token, err := auth(SsoUrl, os.Getenv("CLIENT_ID"), os.Getenv("CLIENT_SECRET"), "client_credentials")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(token.AccessToken)
		authToken = token.AccessToken
	}

	serialNumber := *serialPtr
	if serialNumber == "" {
		log.Fatal("Specify a serial number via the -serial flag")
	}
	if *debugPtr == true {
		debug = true
	}
	_, err := SN2INFO_request(Sn2infoStatusUrl, serialNumber, authToken)
	if err != nil {
		log.Fatal(err)
	}
	_, err = ProductInfo_request(ProductInfoUrl, serialNumber, authToken)
	if err != nil {
		log.Fatal(err)
	}
}
