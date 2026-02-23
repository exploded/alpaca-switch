package server

// ASCOM Alpaca response types

type alpacaResponse struct {
	ClientTransactionID uint32 `json:"ClientTransactionID"`
	ServerTransactionID uint32 `json:"ServerTransactionID"`
	ErrorNumber         int32  `json:"ErrorNumber"`
	ErrorMessage        string `json:"ErrorMessage"`
}

type stringResponse struct {
	alpacaResponse
	Value string `json:"Value"`
}

type booleanResponse struct {
	alpacaResponse
	Value bool `json:"Value"`
}

type int32Response struct {
	alpacaResponse
	Value int32 `json:"Value"`
}

type doubleResponse struct {
	alpacaResponse
	Value float64 `json:"Value"`
}

type stringListResponse struct {
	alpacaResponse
	Value []string `json:"Value"`
}

type uint32ListResponse struct {
	alpacaResponse
	Value []uint32 `json:"Value"`
}

type putResponse struct {
	alpacaResponse
}

// DeviceConfiguration is used in /management/v1/configureddevices.
type DeviceConfiguration struct {
	DeviceName   string `json:"DeviceName"`
	DeviceType   string `json:"DeviceType"`
	DeviceNumber uint32 `json:"DeviceNumber"`
	UniqueID     string `json:"UniqueID"`
}

type managementDevicesListResponse struct {
	alpacaResponse
	Value []DeviceConfiguration `json:"Value"`
}

type managementDescriptionResponse struct {
	alpacaResponse
	Value ServerDescription `json:"Value"`
}

// ServerDescription is used in /management/v1/description.
type ServerDescription struct {
	ServerName          string `json:"ServerName"`
	Manufacturer        string `json:"Manufacturer"`
	ManufacturerVersion string `json:"ManufacturerVersion"`
	Location            string `json:"Location"`
}
