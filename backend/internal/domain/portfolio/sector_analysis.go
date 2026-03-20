package portfolio

// ComputeSectorWeights groups position details by sector and computes
// per-sector aggregate values and weights.  Pure function — no DB calls.
func ComputeSectorWeights(positions []PositionDetail) []SectorWeight {
	sectorMap := make(map[string]*SectorWeight)
	var portfolioTotalValue float64

	for _, pos := range positions {
		portfolioTotalValue += pos.TotalValue

		sectorKey := "UNKNOWN"
		sectorName := "미분류"
		if pos.Sector != nil && *pos.Sector != "" {
			sectorKey = *pos.Sector
		}
		if pos.SectorName != nil && *pos.SectorName != "" {
			sectorName = *pos.SectorName
		}

		sw, ok := sectorMap[sectorKey]
		if !ok {
			sw = &SectorWeight{
				Sector:     sectorKey,
				SectorName: sectorName,
				Positions:  make([]SectorPosition, 0),
			}
			sectorMap[sectorKey] = sw
		}

		sw.TotalValue += pos.TotalValue
		sw.UnrealizedPnL += pos.UnrealizedPnL
		sw.Positions = append(sw.Positions, SectorPosition{
			Symbol:     pos.Symbol,
			SymbolName: pos.SymbolName,
			Value:      pos.TotalValue,
			Weight:     0, // computed below
		})
	}

	results := make([]SectorWeight, 0, len(sectorMap))
	for _, sw := range sectorMap {
		if portfolioTotalValue > 0 {
			sw.Weight = (sw.TotalValue / portfolioTotalValue) * 100
		}
		costBasis := sw.TotalValue - sw.UnrealizedPnL
		if costBasis > 0 {
			sw.UnrealizedPnLPct = (sw.UnrealizedPnL / costBasis) * 100
		}
		// Compute intra-sector position weights
		for i := range sw.Positions {
			if sw.TotalValue > 0 {
				sw.Positions[i].Weight = (sw.Positions[i].Value / sw.TotalValue) * 100
			}
		}
		results = append(results, *sw)
	}

	// Sort by weight descending
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Weight > results[i].Weight {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	return results
}
