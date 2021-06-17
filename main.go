package main

import (
    "io"
    "log"
    "net/http"
    "net/url"
    "os"
    "regexp"
    "strconv"
)

var (
    giteaAddress string
    vReg = regexp.MustCompile(`^(/.*?/.*?)/v\d+`)
)

var hopHeaders = []string{
    "Connection",
    "Keep-Alive",
    "Proxy-Authenticate",
    "Proxy-Authorization",
    "Te", // canonicalized version of "TE"
    "Trailers",
    "Transfer-Encoding",
    "Upgrade",
}

func main() {
    giteaAddress = os.Getenv("GITEA_ADDRESS")
    if giteaAddress == "" {
        log.Fatalln("Environment GITEA_ADDRESS is not defined")
    }
    u, err := url.Parse(giteaAddress)
    if err != nil {
        log.Fatalln("Invalid gitea address ", err)
    }
    server := &proxy{
        Address: u,
    }
    log.Println("Proxy server listen on :3000")
    if err := http.ListenAndServe(":3000", server); err != nil {
        log.Fatal(err)
    }
}

type proxy struct {
    Address     *url.URL
}

func (h *proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    userIP := h.ReadUserIP(r)
    log.Println(userIP, " ", r.Method, " ", r.URL)
    scheme := h.ReadScheme(r)
    if scheme != "http" && scheme != "https" {
       msg := "unsupported protocol scheme " + scheme
       http.Error(w, msg, http.StatusBadRequest)
       log.Println(msg)
       return
    }
    r.RequestURI = ""
    client := &http.Client{}
    h.DeleteHopHeaders(r.Header)
    q := r.URL.Query()
    goGet, _ := strconv.ParseInt(q.Get("go-get"), 10, 64)
    if goGet == 1 {
        r.URL.Path = vReg.ReplaceAllString(r.URL.Path, "$1")
    }
    r.URL = h.Address.ResolveReference(r.URL)

    resp, err := client.Do(r)
    if err != nil {
        http.Error(w, "Server Error", http.StatusInternalServerError)
        log.Println("ServeHTTP: ", err)
        return
    }
    defer resp.Body.Close()

    log.Println(userIP, " ", resp.Status)

    h.DeleteHopHeaders(resp.Header)

    h.CopyHeader(w.Header(), resp.Header)
    w.WriteHeader(resp.StatusCode)
    io.Copy(w, resp.Body)
}

func (h *proxy) CopyHeader(dst, src http.Header) {
    for k, vv := range src {
        for _, v := range vv {
            dst.Add(k, v)
        }
    }
}

func (h *proxy) ReadUserIP(r *http.Request) string {
    IPAddress := r.Header.Get("X-Real-Ip")
    if IPAddress == "" {
        IPAddress = r.Header.Get("X-Forwarded-For")
    }
    if IPAddress == "" {
        IPAddress = r.RemoteAddr
    }
    return IPAddress
}

func (h *proxy) ReadScheme(r *http.Request) string {
    scheme := r.URL.Scheme
    if scheme == "" {
        scheme = r.Header.Get("X-Forwarded-Proto")
    }
    return scheme
}

func (h *proxy) DeleteHopHeaders(header http.Header) {
    for _, h := range hopHeaders {
        header.Del(h)
    }
}