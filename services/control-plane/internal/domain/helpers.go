package domain

func (d Device) HostnameOrID() string {
	if d.DisplayName != "" {
		return d.DisplayName
	}
	if d.Hostname != "" {
		return d.Hostname
	}

	return d.ID
}
