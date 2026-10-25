package main

// contextOccupancyEstimate calibrates the raw JSON-size heuristic against the
// latest provider-measured main-model input. The baseline includes the provider's
// effective prompt overhead, while subsequent growth uses only the local delta.
// A safety margin compensates for estimation drift. Raw estimates remain in use
// until a valid measurement arrives or after the history is structurally reset.
func (a *Agent) contextOccupancyEstimate() int {
	raw := contextEstimate(a.History)
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
	return u.Available && u.InputTokens > 0 && u.EstimatedBefore > 0 && contextEstimate(a.History) >= u.EstimatedBefore
}
