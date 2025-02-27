package internal

// SDK Versioning
var (
	SDKName    = "go"
	SDKVersion = "1.0.0"
)

// ServerInfo contains information about the server environment
type ServerInfo struct {
	Ip        string `json:"ip"`
	Timezone  string `json:"timezone"`
	Software  string `json:"software"`
	Signature string `json:"signature"`
	Protocol  string `json:"protocol"`
	Os        OsInfo `json:"os"`
}

// OsInfo contains information about the operating system
type OsInfo struct {
	Name         string `json:"name"`
	Release      string `json:"release"`
	Architecture string `json:"architecture"`
}

// LanguageInfo contains information about the programming language
type LanguageInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}
