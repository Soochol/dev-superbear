package portfolio

import (
	"math"
	"testing"
)

func taxAlmostEqual(a, b float64) bool {
	return math.Abs(a-b) < 0.01
}

// ---------------------------------------------------------------------------
// KR: 증권거래세 0.18%
// ---------------------------------------------------------------------------

func TestCalculateTax_KR(t *testing.T) {
	// 매도금액 = 15000 * 80 = 1,200,000
	// 거래세 = 1,200,000 * 0.0018 = 2,160
	tax := CalculateTax(MarketKR, 15000, 80, 400000)
	expected := 1200000.0 * KRTransactionTaxRate
	if !taxAlmostEqual(tax, expected) {
		t.Errorf("KR tax: expected %f, got %f", expected, tax)
	}
}

func TestCalculateTax_KR_Loss(t *testing.T) {
	// KR: 증권거래세는 손익 무관하게 매도금액 기준으로 부과
	tax := CalculateTax(MarketKR, 8000, 100, -200000)
	expected := 800000.0 * KRTransactionTaxRate
	if !taxAlmostEqual(tax, expected) {
		t.Errorf("KR tax on loss: expected %f, got %f", expected, tax)
	}
}

// ---------------------------------------------------------------------------
// US: 양도소득세 22%
// ---------------------------------------------------------------------------

func TestCalculateTax_US_Gain(t *testing.T) {
	// grossPnL = 500,000
	// tax = 500,000 * 0.22 = 110,000
	tax := CalculateTax(MarketUS, 150, 100, 500000)
	expected := 500000.0 * USCapitalGainsTaxRate
	if !taxAlmostEqual(tax, expected) {
		t.Errorf("US tax: expected %f, got %f", expected, tax)
	}
}

func TestCalculateTax_US_Loss(t *testing.T) {
	// 손실 거래 → 세금 없음
	tax := CalculateTax(MarketUS, 100, 50, -50000)
	if tax != 0 {
		t.Errorf("US tax on loss: expected 0, got %f", tax)
	}
}

// ---------------------------------------------------------------------------
// CalculateTaxDetailed
// ---------------------------------------------------------------------------

func TestCalculateTaxDetailed_KR(t *testing.T) {
	r := CalculateTaxDetailed(MarketKR, 1200000, 400000)
	expected := 1200000.0 * KRTransactionTaxRate
	if !taxAlmostEqual(r.TransactionTax, expected) {
		t.Errorf("expected transactionTax %f, got %f", expected, r.TransactionTax)
	}
	if r.IncomeTax != 0 {
		t.Errorf("expected incomeTax 0 for KR, got %f", r.IncomeTax)
	}
	if !taxAlmostEqual(r.TotalTax, expected) {
		t.Errorf("expected totalTax %f, got %f", expected, r.TotalTax)
	}
}

func TestCalculateTaxDetailed_US(t *testing.T) {
	r := CalculateTaxDetailed(MarketUS, 1500000, 500000)
	expected := 500000.0 * USCapitalGainsTaxRate
	if r.TransactionTax != 0 {
		t.Errorf("expected transactionTax 0 for US, got %f", r.TransactionTax)
	}
	if !taxAlmostEqual(r.IncomeTax, expected) {
		t.Errorf("expected incomeTax %f, got %f", expected, r.IncomeTax)
	}
}

// ---------------------------------------------------------------------------
// Annual US tax with basic deduction
// ---------------------------------------------------------------------------

func TestCalculateAnnualUSTax_AboveDeduction(t *testing.T) {
	// 총 양도차익 500만원, 기본공제 250만원 → 과세분 250만원 * 22% = 55만원
	r := CalculateAnnualUSTax(5000000)
	taxable := 5000000.0 - USBasicDeduction // 2,500,000
	expected := taxable * USCapitalGainsTaxRate // 550,000
	if !taxAlmostEqual(r.IncomeTax, expected) {
		t.Errorf("expected annual US tax %f, got %f", expected, r.IncomeTax)
	}
}

func TestCalculateAnnualUSTax_BelowDeduction(t *testing.T) {
	r := CalculateAnnualUSTax(2000000) // 200만 < 250만 공제
	if r.TotalTax != 0 {
		t.Errorf("expected 0 tax below deduction, got %f", r.TotalTax)
	}
}

func TestCalculateAnnualUSTax_Negative(t *testing.T) {
	r := CalculateAnnualUSTax(-1000000) // 손실
	if r.TotalTax != 0 {
		t.Errorf("expected 0 tax for loss, got %f", r.TotalTax)
	}
}
