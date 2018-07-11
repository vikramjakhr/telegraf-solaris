package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	"regexp"
	"strconv"
)

// JBoss the main collectod struct
type JBoss4 struct {
	Servers []string
	Metrics []string

	Username string
	Password string

	ExecAsDomain bool `toml:"exec_as_domain"`

	Authorization string

	ResponseTimeout Duration

	// Path to CA file
	SSLCA string `toml:"ssl_ca"`
	// Path to host cert file
	SSLCert string `toml:"ssl_cert"`
	// Path to cert key file
	SSLKey string `toml:"ssl_key"`
	// Use SSL but skip chain & host verification
	InsecureSkipVerify bool

	client HTTPClient
}

var jboss4SampleConfig = `
  # Config for get statistics from JBoss4 AS
  servers = [
    "http://[jboss-server-ip]:<port>/web-console/ServerInfo.jsp",
  ]
	## Execution Mode
	exec_as_domain = false
  ## Username and password
  username = ""
  password = ""
	## authorization mode could be "basic" or "digest"
  authorization = "digest"

  ## Optional SSL Config
  # ssl_ca = "/etc/telegraf/ca.pem"
  # ssl_cert = "/etc/telegraf/cert.pem"
  # ssl_key = "/etc/telegraf/key.pem"
  ## Use SSL but skip chain & host verification
  # insecure_skip_verify = false
	## Metric selection
	metrics =[
		"jvm",
		"web_con",
		"database",
	]
`

// SampleConfig returns a sample configuration block
func (*JBoss4) SampleConfig() string {
	return jboss4SampleConfig
}

// Description just returns a short description of the JBoss4 plugin
func (*JBoss4) Description() string {
	return "Telegraf plugin for gathering metrics from JBoss4 AS"
}

// Gather Gathers data for all servers.
func (h *JBoss4) Gather(acc Accumulator) error {
	var wg sync.WaitGroup

	if h.ResponseTimeout.Duration < time.Second {
		h.ResponseTimeout.Duration = time.Second * 5
	}

	if h.client.HTTPClient() == nil {
		tlsCfg, err := GetTLSConfig(
			h.SSLCert, h.SSLKey, h.SSLCA, h.InsecureSkipVerify)
		if err != nil {
			return err
		}
		tr := &http.Transport{
			ResponseHeaderTimeout: time.Duration(3 * time.Second),
			TLSClientConfig:       tlsCfg,
		}
		client := &http.Client{
			Transport: tr,
			Timeout:   h.ResponseTimeout.Duration,
		}
		h.client.SetHTTPClient(client)
	}

	errorChannel := make(chan error, len(h.Servers))

	for _, server := range h.Servers {
		wg.Add(1)
		go func(server string) {
			defer wg.Done()
			//default as standalone server
			hosts := HostResponse{Outcome: "", Result: []string{"standalone"}}

			h.getServersOnHost(acc, server, hosts.Result)

		}(server)
	}

	wg.Wait()
	close(errorChannel)

	// Get all errors and return them as one giant error
	errorStrings := []string{}
	for err := range errorChannel {
		errorStrings = append(errorStrings, err.Error())
	}

	if len(errorStrings) == 0 {
		return nil
	}
	return errors.New(strings.Join(errorStrings, "\n"))
}

// Gathers data from a particular host
// Parameters:
//     acc      : The telegraf Accumulator to use
//     serverURL: endpoint to send request to
//     host     : the host being queried
//
// Returns:
//     error: Any error that may have occurred

func (h *JBoss4) getServersOnHost(
	acc Accumulator,
	serverURL string,
	hosts []string,
) error {
	var wg sync.WaitGroup

	errorChannel := make(chan error, len(hosts))

	for _, host := range hosts {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()
			log.Printf("I! Get Servers from host: %s\n", host)

			servers := HostResponse{Outcome: "", Result: []string{"standalone"}}

			for _, server := range servers.Result {
				log.Printf("I! JBoss4 Plugin Processing Servers from host:[ %s ] : Server [ %s ]\n", host, server)
				for _, v := range h.Metrics {
					switch v {
					case "jvm":
						h.getJVMStatistics(acc, serverURL, host, server)
					default:
						log.Printf("E! Jboss doesn't exist the metric set %s\n", v)
					}
				}
			}
		}(host)
	}

	wg.Wait()
	close(errorChannel)

	// Get all errors and return them as one giant error
	errorStrings := []string{}
	for err := range errorChannel {
		errorStrings = append(errorStrings, err.Error())
	}

	if len(errorStrings) == 0 {
		return nil
	}
	return errors.New(strings.Join(errorStrings, "\n"))
}

func (h *JBoss4) getJVMStatistics(
	acc Accumulator,
	serverURL string,
	host string,
	serverName string,
) error {

	out, err := h.doRequest(serverURL)

	log.Printf("I! JBoss4 API Req err: %s", err)

	if err != nil {
		return fmt.Errorf("E! Error on request to %s : %s\n", serverURL, err)
	}

	respHtml := string(out[:])
	fields := parseHtml(respHtml)
	if len(fields) > 0 {
		log.Printf("I! Jboss4 response fields: %v", fields)

		tags := map[string]string{
			"jboss_host":   host,
			"jboss_server": serverURL,
		}
		acc.AddFields("jboss_jvm", fields, tags)
	}
	return nil
}

type JVMStats struct {
}

func parseHtml(html string) (map[string]interface{}) {
	fields := make(map[string]interface{})

	// Max Memory
	regex, _ := regexp.Compile("Max Memory: </b>")
	lastLoc := regex.FindStringIndex(html)[1]
	str := html[lastLoc:lastLoc+10]
	startIdx := strings.LastIndex(str, " MB")
	valueStr := str[:startIdx]
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		log.Printf("E! Error while parsing jboss4 response html: %s", err)
		return fields
	}
	fields["heap_max"] = convertToBytes(value)

	// Free Memory
	regex, _ = regexp.Compile("Free Memory: </b>")
	lastLoc = regex.FindStringIndex(html)[1]
	str = html[lastLoc:lastLoc+10]
	startIdx = strings.LastIndex(str, " MB")
	valueStr = str[:startIdx]
	value, err = strconv.ParseFloat(valueStr, 64)
	if err != nil {
		log.Printf("E! Error while parsing jboss4 response html: %s", err)
		return fields
	}
	fields["heap_free"] = convertToBytes(value)

	// Total Memory
	regex, _ = regexp.Compile("Total Memory: </b>")
	lastLoc = regex.FindStringIndex(html)[1]
	str = html[lastLoc:lastLoc+10]
	startIdx = strings.LastIndex(str, " MB")
	valueStr = str[:startIdx]
	value, err = strconv.ParseFloat(valueStr, 64)
	if err != nil {
		log.Printf("E! Error while parsing jboss4 response html: %s", err)
		return fields
	}
	fields["heap_committed"] = convertToBytes(value)

	// #Threads:
	regex, _ = regexp.Compile("#Threads: </b>")
	lastLoc = regex.FindStringIndex(html)[1]
	str = html[lastLoc:lastLoc+10]
	startIdx = strings.LastIndex(str, "</font>")
	valueStr = str[:startIdx]
	value, err = strconv.ParseFloat(valueStr, 64)
	if err != nil {
		log.Printf("E! Error while parsing jboss4 response html: %s", err)
		return fields
	}
	fields["thread-count"] = value

	return fields
}

func convertToBytes(value float64) float64 {
	if value > 0 {
		return value * 1024 * 1024
	}
	return 0
}

func (j *JBoss4) doRequest(domainUrl string) ([]byte, error) {

	serverUrl, err := url.Parse(domainUrl)
	if err != nil {
		return nil, err
	}
	method := "GET"

	req, err := http.NewRequest(method, serverUrl.String(), nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(j.Username, j.Password)

	resp, err := j.client.MakeRequest(req)
	if err != nil {
		log.Printf("D! HTTP REQ:%#+v", req)
		log.Printf("D! HTTP RESP:%#+v", resp)
		return nil, err
	}
	defer resp.Body.Close()

	log.Printf("D! JBoss4 API Req HTTP REQ:%#+v", req)
	log.Printf("D! JBoss4 API Req HTTP RESP:%#+v", resp)

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("Response from url \"%s\" has status code %d (%s), expected %d (%s)",
			req.RequestURI,
			resp.StatusCode,
			http.StatusText(resp.StatusCode),
			http.StatusOK,
			http.StatusText(http.StatusOK))
		return nil, err
	}

	// read body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("E! JBoss4 Error: %s", err)
		return nil, err
	}

	// Debug response
	//log.Printf("D! body: %s\n", body)

	return []byte(body), nil
}
