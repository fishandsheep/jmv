package okm

import "time"

const (
	DefaultMirror = "https://mirrors.tuna.tsinghua.edu.cn/Adoptium"
	Version       = "0.1.0"
)

type Runtime string

const (
	RuntimeJDK Runtime = "jdk"
	RuntimeJRE Runtime = "jre"
)

type Platform struct {
	Arch string
	OS   string
	Ext  string
}

type Release struct {
	Runtime  Runtime
	Major    string
	FileName string
	URL      string
	Platform Platform
}

type Metadata struct {
	Runtime     Runtime   `json:"runtime"`
	Major       string    `json:"major"`
	FileName    string    `json:"fileName"`
	URL         string    `json:"url"`
	Platform    string    `json:"platform"`
	Home        string    `json:"home"`
	InstalledAt time.Time `json:"installedAt"`
}

type Current struct {
	Runtime Runtime `json:"runtime"`
	Major   string  `json:"major"`
	Home    string  `json:"home"`
}
