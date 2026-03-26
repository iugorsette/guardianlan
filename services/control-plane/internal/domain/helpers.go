package domain

func (d Device) HostnameOrID() string {
	if d.Hostname != "" {
		return d.Hostname
	}

	return d.ID
}

