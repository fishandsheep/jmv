package okm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMirrorListJDKAndJRE(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/Adoptium/", func(w http.ResponseWriter, r *http.Request) {
		requireDownloadHeaders(t, w, r)
		w.Write([]byte(`<a href="8/">8/</a><a href="17/">17/</a>`))
	})
	mux.HandleFunc("/Adoptium/8/jdk/x64/linux/", func(w http.ResponseWriter, r *http.Request) {
		requireDownloadHeaders(t, w, r)
		w.Write([]byte(`<a href="OpenJDK8U-jdk_x64_linux_hotspot_8u482b08.tar.gz">jdk</a>`))
	})
	mux.HandleFunc("/Adoptium/17/jdk/x64/linux/", func(w http.ResponseWriter, r *http.Request) {
		requireDownloadHeaders(t, w, r)
		w.Write([]byte(`<a href="OpenJDK17U-jdk_x64_linux_hotspot_17.0.19_10.tar.gz">jdk</a>`))
	})
	mux.HandleFunc("/Adoptium/8/jre/x64/linux/", func(w http.ResponseWriter, r *http.Request) {
		requireDownloadHeaders(t, w, r)
		w.Write([]byte(`<a href="OpenJDK8U-jre_x64_linux_hotspot_8u482b08.tar.gz">jre</a>`))
	})
	mux.HandleFunc("/Adoptium/17/jre/x64/linux/", func(w http.ResponseWriter, r *http.Request) {
		requireDownloadHeaders(t, w, r)
		w.Write([]byte(`<a href="OpenJDK17U-jre_x64_linux_hotspot_17.0.19_10.tar.gz">jre</a>`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewMirrorClient(server.URL + "/Adoptium")
	platform := Platform{Arch: "x64", OS: "linux", Ext: ".tar.gz"}

	jdks, err := client.List(context.Background(), RuntimeJDK, platform)
	if err != nil {
		t.Fatal(err)
	}
	if len(jdks) != 2 || jdks[0].Major != "8" || jdks[1].Major != "17" {
		t.Fatalf("unexpected JDK releases: %#v", jdks)
	}

	jre, err := client.Resolve(context.Background(), RuntimeJRE, "17", platform)
	if err != nil {
		t.Fatal(err)
	}
	if jre.FileName != "OpenJDK17U-jre_x64_linux_hotspot_17.0.19_10.tar.gz" {
		t.Fatalf("unexpected JRE filename: %s", jre.FileName)
	}
}

func TestMirrorIgnoresMacPkg(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/Adoptium/21/jdk/aarch64/mac/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`
			<a href="OpenJDK21U-jdk_aarch64_mac_hotspot_21.0.10_7.pkg">pkg</a>
			<a href="OpenJDK21U-jdk_aarch64_mac_hotspot_21.0.10_7.tar.gz">tar</a>
		`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	release, err := NewMirrorClient(server.URL+"/Adoptium").Resolve(
		context.Background(),
		RuntimeJDK,
		"21",
		Platform{Arch: "aarch64", OS: "mac", Ext: ".tar.gz"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if release.FileName != "OpenJDK21U-jdk_aarch64_mac_hotspot_21.0.10_7.tar.gz" {
		t.Fatalf("unexpected filename: %s", release.FileName)
	}
}

func requireDownloadHeaders(t *testing.T, w http.ResponseWriter, r *http.Request) {
	t.Helper()
	if r.Header.Get("User-Agent") != userAgent || r.Header.Get("Accept") != "*/*" {
		http.Error(w, "forbidden", http.StatusForbidden)
	}
}
