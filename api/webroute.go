package api

// WebRoute is a single public endpoint route for a service.
type WebRoute struct {
	Provider ProviderName `json:"provider"`

	From Address  `json:"from"`
	To   *Address `json:"to"`
}

type Address struct {
	// Address is a full host and path value used for direct HTTP calls.
	Address string `json:"address"`
	// Domain is a public domain where service is available.
	Domain string `json:"domain"`
	// Port is a service container port exposed by a reverse proxy.
	Port string `json:"port"`
}
