package parser

import (
	"encoding/json"
	"testing"
)

func TestAPIResponseAcceptsStringAddressName(t *testing.T) {
	body := []byte(`{
		"meta": {"code": 200},
		"result": {
			"total": 1,
			"items": [{
				"id": "1",
				"name": "Wipon",
				"address_name": "Turan avenue, 19/2",
				"full_address_name": "Astana, Turan avenue, 19/2"
			}]
		}
	}`)

	var response apiResponse
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if got := response.Result.Items[0].AddressName; got != "Turan avenue, 19/2" {
		t.Fatalf("AddressName = %q", got)
	}
}

func TestExtractContactsReadsPhoneAndWebsite(t *testing.T) {
	groups := []apiContactGroup{
		{
			Contacts: []apiContact{
				{Type: "phone", Value: "+77172123456"},
				{Type: "website", URL: "https://example.com"},
			},
		},
	}

	phone, website := extractContacts(groups)
	if phone != "+77172123456" {
		t.Fatalf("phone = %q", phone)
	}
	if website != "https://example.com" {
		t.Fatalf("website = %q", website)
	}
}

func TestExtractContactsUsesFallbackFields(t *testing.T) {
	groups := []apiContactGroup{
		{
			Contacts: []apiContact{
				{Type: "phone", Text: "+7 7172 12 34 56"},
				{Type: "website", Value: "example.kz"},
			},
		},
	}

	phone, website := extractContacts(groups)
	if phone != "+7 7172 12 34 56" {
		t.Fatalf("phone = %q", phone)
	}
	if website != "example.kz" {
		t.Fatalf("website = %q", website)
	}
}
