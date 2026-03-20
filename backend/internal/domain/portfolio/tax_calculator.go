package portfolio

// ---------------------------------------------------------------------------
// Market enum
// ---------------------------------------------------------------------------

// Market distinguishes domestic (KR) from overseas (US) exchanges.
type Market string

const (
	MarketKR Market = "KR"
	MarketUS Market = "US"
)

// ---------------------------------------------------------------------------
// Tax configuration constants
// ---------------------------------------------------------------------------

const (
	// KR: 증권거래세 0.18%  (일반 투자자 양도세 비과세)
	KRTransactionTaxRate = 0.0018

	// US: 해외 양도소득세 22%, 기본공제 250만원/년
	USCapitalGainsTaxRate = 0.22
	USBasicDeduction      = 2_500_000 // 250만원 (연간 합산 시 적용)
)

// ---------------------------------------------------------------------------
// TaxResult
// ---------------------------------------------------------------------------

// TaxResult contains the breakdown of taxes for a single trade event.
type TaxResult struct {
	TransactionTax float64 // 증권거래세 (KR only)
	IncomeTax      float64 // 양도소득세 (US only, or KR 대주주)
	TotalTax       float64 // sum
}

// ---------------------------------------------------------------------------
// CalculateTax — per-trade tax
// ---------------------------------------------------------------------------

// CalculateTax computes the tax for a single sell leg.
//
// For KR: 증권거래세 = 매도금액 * 0.18%.  일반 투자자 양도세 없음.
// For US: 양도차익의 22%.  기본공제는 연간 합산 시 적용하므로 여기서는 미적용.
//         손실 거래(grossPnL <= 0)에는 세금 없음.
func CalculateTax(market Market, sellPrice float64, sellQty int, grossPnL float64) float64 {
	switch market {
	case MarketKR:
		return sellPrice * float64(sellQty) * KRTransactionTaxRate
	case MarketUS:
		if grossPnL <= 0 {
			return 0
		}
		return grossPnL * USCapitalGainsTaxRate
	default:
		return 0
	}
}

// CalculateTaxDetailed returns a full TaxResult breakdown.
func CalculateTaxDetailed(market Market, sellAmount float64, grossPnL float64) TaxResult {
	switch market {
	case MarketKR:
		txTax := sellAmount * KRTransactionTaxRate
		return TaxResult{
			TransactionTax: txTax,
			IncomeTax:      0,
			TotalTax:       txTax,
		}
	case MarketUS:
		if grossPnL <= 0 {
			return TaxResult{}
		}
		incomeTax := grossPnL * USCapitalGainsTaxRate
		return TaxResult{
			TransactionTax: 0,
			IncomeTax:      incomeTax,
			TotalTax:       incomeTax,
		}
	default:
		return TaxResult{}
	}
}

// CalculateAnnualUSTax computes US capital gains tax with the annual basic
// deduction (250만원) applied. Use this for year-end tax simulation.
func CalculateAnnualUSTax(totalGain float64) TaxResult {
	if totalGain <= 0 {
		return TaxResult{}
	}
	taxable := totalGain - USBasicDeduction
	if taxable <= 0 {
		return TaxResult{}
	}
	incomeTax := taxable * USCapitalGainsTaxRate
	return TaxResult{
		TransactionTax: 0,
		IncomeTax:      incomeTax,
		TotalTax:       incomeTax,
	}
}
