package networkcommon

type IPInfoResponse struct {
	IP       string `json:"ip"`
	Label    string `json:"label,omitempty"`
	ASN      uint   `json:"asn"`
	Org      string `json:"org"`
	Country  string `json:"country"`
	City     string `json:"city"`
	Location string `json:"location"`
}
