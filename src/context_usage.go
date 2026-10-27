package main

// rawModelContextEstimate is the local heuristic for charter + recent tail + budget overhead.
func (a *Agent) rawModelContextEstimate() int {
	if a == nil {
		return budgetBlockOverhead
	}
	return contextEstimate(a.modelInputCore()) + budgetBlockOverhead
}

// contextOccupancyEstimate calibrates the raw JSON-size heuristic against the
// latest provider-measured main-model input. The baseline includes the provider's
// effective prompt overhead, while subsequent growth uses only the local delta.
func (a *Agent) contextOccupancyEstimate() int {
	raw := a.rawModelContextEstimate()
	if !a.contextUsageCalibrated() {
		return raw
	}
	u := a.Client.LastMainUsage
	growth := raw - u.EstimatedBefore
	if growth < 0 {
		return raw
	}
	measured := int(u.InputTokens) + growth
	// Reserve 15% for drift in token estimation and hidden request overhead.
	return measured + (measured+6)/7
}

func (a *Agent) contextUsageCalibrated() bool {
	if a == nil || a.Client == nil {
		return false
	}
	u := a.Client.LastMainUsage
	raw := a.rawModelContextEstimate()
	return u.Available && u.InputTokens > 0 && u.EstimatedBefore > 0 && raw >= u.EstimatedBefore
}
