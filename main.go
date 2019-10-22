package main

import (
	"strconv"
	"strings"
	"bytes"
	"fmt"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"encoding/json"
	"os/exec"
	"net/http"
	"github.com/gorilla/mux"
)

type ShadowsocksDataPort struct {
	ServerPort int    `json:"server_port"`
	Password   string `json:"password,omitempty"`
}

type ShadowsocksException struct {
	Status string `json:"status"`
	Reason string `json:"error"`
}

var shadowsocksManagerDomain string
var shadowsocksManagerPort int

func trimLeftChars(s string, n int) string {
	m := 0
	for i := range s {
		if m >= n {
			return s[i:]
		}
		m++
	}

	return s[:0]
}

func ss_cast_json(status string, reason string) string {
	ssException := &ShadowsocksException{
		Status: status,
		Reason: reason,
	}

	jsonBytes, err := json.Marshal(ssException)
	if err != nil {
		log.Println(err)
		return `{"status": "internal_error", "reason": "cast_exception_failure"}`
	}

	return string(jsonBytes)
}

func ss_manager_get_traffic_statistics(domain string, port int) (string, error) {
	result, err := execNc("ping", domain, port)
	if err != nil {
		return ss_cast_json("comm_error", "stat_comm_failure"), err
	}

	if strings.HasPrefix(result, "stat: ") {
		return trimLeftChars(result, 6), nil
	} else {
		return ss_cast_json("unexpected_answer", result), nil
	}
}

func ss_manager_add_port(userPort int, userPassword string, domain string, port int) (string, error) {
	dataPort := &ShadowsocksDataPort{
		ServerPort: userPort,
		Password: userPassword,
	}

	payloadBytes, err := json.Marshal(dataPort)
	if err != nil {
		return ss_cast_json("internal_error", "add_port_json_marshal_failure"), err
	}

	payload := "add: " + string(payloadBytes)
	result, err := execNc(payload, domain, port)
	if err != nil {
		return ss_cast_json("comm_error", "add_port_comm_failure"), err
	}

	if result == "ok" {
		return ss_cast_json("ok", ""), nil
	} else if result == "port is not available" {
		return ss_cast_json("ok", "port_already_exists"), nil
	} else {
		return ss_cast_json("unexpected_answer", result), nil
	}
}

func ss_manager_remove_port(userPort int, domain string, port int) (string, error) {
	dataPort := &ShadowsocksDataPort{
		ServerPort: userPort,
	}

	payloadBytes, err := json.Marshal(dataPort)
	if err != nil {
		return ss_cast_json("internal_error", "remove_port_json_marshal_failure"), err
	}

	payload := "remove: " + string(payloadBytes)
	result, err := execNc(payload, domain, port)
	if err != nil {
		return ss_cast_json("comm_error", "remove_port_comm_failure"), err
	}

	if result == "ok" {
		return ss_cast_json("ok", ""), nil
	} else {
		return ss_cast_json("unexpected_answer", result), nil
	}
}

func execNc(payload string, domain string, port int) (string, error) {
	cmd := exec.Command("nc", "-u", "-w1", domain, strconv.Itoa(port))
	
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return `{"error": "nc_input_failure"}`, err
	}

	go func() {
		defer stdin.Close()
		io.WriteString(stdin, payload)
	}()

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	err = cmd.Run()
	if err != nil {
		return `{"error": "nc_run_failure"}`, err
	}

	return stdout.String(), nil
}

func ss_api_healthcheck(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "ok")
}

func ss_api_statistics(w http.ResponseWriter, r *http.Request) {
	result, err := ss_manager_get_traffic_statistics(shadowsocksManagerDomain, shadowsocksManagerPort)
	if err != nil {
		log.Println(err)
	}

	fmt.Fprintf(w, result)
}

func ss_api_add_port(w http.ResponseWriter, r *http.Request) {
	portNumberStr := mux.Vars(r)["portNumber"]
	portNumber, err := strconv.Atoi(portNumberStr)
	if err != nil {
		fmt.Fprintf(w, ss_cast_json("invalid_port", "not_a_port_number"))
	}

	requestBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Fprintf(w, ss_cast_json("invalid_password", "port_password_invalid_or_undefined"))
	}

	w.WriteHeader(http.StatusCreated)
	result, err := ss_manager_add_port(portNumber, string(requestBody), shadowsocksManagerDomain, shadowsocksManagerPort)
	if err != nil {
		log.Println(err)
	}

	fmt.Fprintf(w, result)
}

func ss_api_remove_port(w http.ResponseWriter, r *http.Request) {
	portNumberStr := mux.Vars(r)["portNumber"]
        portNumber, err := strconv.Atoi(portNumberStr)
        if err != nil {
                fmt.Fprintf(w, ss_cast_json("invalid_port", "not_a_port_number"))
        }

	w.WriteHeader(http.StatusCreated)
	result, err := ss_manager_remove_port(portNumber, shadowsocksManagerDomain, shadowsocksManagerPort)
	if err != nil {
		log.Println(err)
	}

	fmt.Fprintf(w, result)
}

func main() {
	appName := "Shadowsocks API Server"
	appVersion := "1.0.0.0"
	appAuthor := "2019 Snooky Booger"

	domainPtr := flag.String("hostname", "localhost", "Shadowsocks manager domain name or IP")
	portPtr := flag.Int("port", 43456, "Shadowsocks manager port")
	listenPortPtr := flag.Int("listen", 8080, "Listening port")

	flag.Usage = func() {
		fmt.Printf("%s\n", appName)
		fmt.Printf("Version %s\n", appVersion)
		fmt.Printf("Copyright (c) %s. All rights reserved.\n", appAuthor)
		fmt.Printf("\n")
		fmt.Printf("Parameters:\n")
		flag.PrintDefaults()
		fmt.Printf("\n")
		fmt.Printf("Examples:\n")
		fmt.Printf("  %s\n", "curl http://localhost:8080/healthcheck")
		fmt.Printf("  %s\n", "curl http://localhost:8080/ports")
		fmt.Printf("  %s\n", "curl -X POST http://localhost:8080/ports/12345 -d MyPassword")
		fmt.Printf("  %s\n", "curl -X DELETE http://localhost:8080/ports/12345")
		fmt.Printf("\n")
	}

	flag.Parse()

	log.Println(appName)
	log.Println("Version", appVersion)
	log.Println("Copyright (c) " + appAuthor + ". All rights reserved.")
	log.Println("")

        shadowsocksManagerDomain = *domainPtr
        shadowsocksManagerPort = *portPtr

	log.Println("Upstream Shadowsocks manager:")
	log.Println("  Hostname ", shadowsocksManagerDomain)
	log.Println("  Port     ", shadowsocksManagerPort)
	log.Println("")

	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/healthcheck", ss_api_healthcheck)
	router.HandleFunc("/ports", ss_api_statistics).Methods("GET")
	router.HandleFunc("/ports/{portNumber}", ss_api_add_port).Methods("POST")
	router.HandleFunc("/ports/{portNumber}", ss_api_remove_port).Methods("DELETE")

	listenPort := *listenPortPtr
	log.Println("Listening on 0.0.0.0:", listenPort)
	log.Fatal(http.ListenAndServe(":" + strconv.Itoa(listenPort), router))
}

