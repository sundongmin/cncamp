package main

import (
	"flag"
	"fmt"
	"github.com/cncamp/httpserver/middleware"
	"io"
	"k8s.io/klog/v2"
	"net"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
)

func main() {
	flag.Set("alsologtostderr", "true")
	flag.Parse()

	klogFlags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(klogFlags)

	middlewareChain := middleware.MiddlewareChain{
		NewClientIpInterceptor(),
		NewResponseCodeInterceptor(),
	}

	mux := http.NewServeMux()
	mux.Handle("/", middlewareChain.Handler(rootHandler))

	mux.HandleFunc("/healthz", healthz)
	err := http.ListenAndServe(":80", mux)
	if err != nil {
		klog.Fatal(err)
	}
}

func healthz(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "200\n")
}

func rootHandler(w http.ResponseWriter, r *http.Request) {

	io.WriteString(w, "===================Details of the http request header:============\n")

	max := 0
	for k := range r.Header {
		if l := len(k); l > max {
			max = l
		}
	}

	for k, v := range r.Header {
		io.WriteString(w, fmt.Sprintf("%-"+strconv.Itoa(max)+"s = %-s\n", k, v))
	}

	io.WriteString(w, fmt.Sprintf("Version = %s\n", getEnv()))
}

func getEnv() string {
	return os.Getenv("VERSION")
}

func NewClientIpInterceptor() middleware.MiddlewareInterceptor {
	return func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		ip := ClientIP(r)
		klog.Infoln("客户端ip为: %s", ip)
		next(w, r)
	}
}

func NewResponseCodeInterceptor() middleware.MiddlewareInterceptor {
	return func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {

		next(w, r)
		value := reflect.ValueOf(w).Elem()
		klog.Infoln("返回状态码为: %d", value.FieldByName("status"))
	}
}

func ClientIP(r *http.Request) string {
	xForwardedFor := r.Header.Get("X-Forwarded-For")
	ip := strings.TrimSpace(strings.Split(xForwardedFor, ",")[0])
	if ip != "" {
		return ip
	}

	ip = strings.TrimSpace(r.Header.Get("X-Real-Ip"))
	if ip != "" {
		return ip
	}

	if ip, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr)); err == nil {
		return ip
	}

	return "unknown"
}
