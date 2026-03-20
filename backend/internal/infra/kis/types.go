package kis

import "time"

type AuthToken struct {
	AccessToken string
	TokenType   string
	ExpiresAt   time.Time
}

type KISCandle struct {
	StckBsopDate string `json:"stck_bsop_date"`
	StckOprc     string `json:"stck_oprc"`
	StckHgpr     string `json:"stck_hgpr"`
	StckLwpr     string `json:"stck_lwpr"`
	StckClpr     string `json:"stck_clpr"`
	AcmlVol      string `json:"acml_vol"`
	AcmlTrPbmn   string `json:"acml_tr_pbmn"`
}

type KISPriceResponse struct {
	StckPrpr   string `json:"stck_prpr"`
	PrdyVrss   string `json:"prdy_vrss"`
	PrdyCtrt   string `json:"prdy_ctrt"`
	AcmlVol    string `json:"acml_vol"`
	Per        string `json:"per"`
	Eps        string `json:"eps"`
	HtsKorIsnm string `json:"hts_kor_isnm"`
}

type NormalizedCandle struct {
	Time   string  `json:"time"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume int64   `json:"volume"`
}

type CandleResponse struct {
	Output2 []KISCandle `json:"output2"`
}

type PriceResponse struct {
	Output *KISPriceResponse `json:"output"`
}
