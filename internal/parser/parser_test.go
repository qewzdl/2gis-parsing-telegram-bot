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
