package dart

type RawFinancials struct {
	Revenue         string `json:"revenue"`
	OperatingProfit string `json:"operating_profit"`
	NetIncome       string `json:"net_income"`
}

type NormalizedFinancials struct {
	Revenue         *float64 `json:"revenue"`
	OperatingProfit *float64 `json:"operatingProfit"`
	NetMargin       *float64 `json:"netMargin"`
	PER             *float64 `json:"per"`
	PBR             *float64 `json:"pbr"`
	ROE             *float64 `json:"roe"`
}

type DARTFinancialResponse struct {
	Status  string              `json:"status"`
	Message string              `json:"message"`
	List    []DARTFinancialItem `json:"list"`
}

type DARTFinancialItem struct {
	RceptNo   string `json:"rcept_no"`
	BsnsYear  string `json:"bsns_year"`
	CorpCode  string `json:"corp_code"`
	AccountNm string `json:"account_nm"`
	ThstrmAmt string `json:"thstrm_amount"`
}
