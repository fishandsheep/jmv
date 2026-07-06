package jmv

import "time"

const (
	DefaultMirror      = "https://mirrors.tuna.tsinghua.edu.cn/Adoptium"
	DefaultMavenMirror = "https://mirrors.aliyun.com/apache/maven"
	Version            = "0.1.0"
)

type Runtime string

const (
	RuntimeJDK   Runtime = "jdk"
	RuntimeJRE   Runtime = "jre"
	RuntimeMaven Runtime = "maven"
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
	SHA256   string
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
