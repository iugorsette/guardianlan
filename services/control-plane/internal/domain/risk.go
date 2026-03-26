package domain

func ScoreDevice(event DeviceEvent) int {
	score := 10

	if event.Vendor == "" {
		score += 10
	}

	if event.MAC == "" {
		score += 10
	}

	if len(event.IPs) > 1 {
		score += 5
	}

	switch event.DeviceType {
	case "camera":
		score += 30
	case "iot":
		score += 20
	case "tablet":
		score += 10
	default:
		score += 5
	}

	if !event.Managed {
		score += 15
	}

	if score > 100 {
		return 100
	}

	return score
}

