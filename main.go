package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var sink float64

type Config struct {
	Port             string
	MinStartDelayMs  int
	MaxStartDelayMs  int
	MinDelayMs       int
	MaxDelayMs       int
	BurnCPU          bool
	CPUComplexity    int
	ExternalServices []string
	MaxCallDepth     int
	RequestTimeout   int
	Hostname         string
}

type HealthResponse struct {
	StatusCode            int               `json:"status_code"`
	ModeActive            string            `json:"mode_active"`
	DepthLevel            int               `json:"depth_level"`
	ReachedLimit          bool              `json:"reached_limit"`
	CPUTime               string            `json:"cpu_time"`
	WaitTime              string            `json:"wait_time"`
	TotalTime             string            `json:"total_time"`
	ServicesCalled        map[string]string `json:"services_called,omitempty"`
	ApplicationAttributes string            `json:"application_attributes,omitempty"`
}

func parseEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func parseEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
	}
	return defaultVal
}

func parseEnvString(key string, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func loadHostname() string {
	h, err := os.Hostname()
	if err != nil {
		h = "unknown"
	}
	return parseEnvString("HOSTNAME", h)
}

func loadConfig() Config {
	port := parseEnvString("PORT", "8080")
	extSvcStr := parseEnvString("EXTERNAL_SERVICES", "")
	var extSvc []string
	if extSvcStr != "" {
		parts := strings.Split(extSvcStr, ",")
		for _, p := range parts {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				extSvc = append(extSvc, trimmed)
			}
		}
	}

	return Config{
		Port:             port,
		MinStartDelayMs:  parseEnvInt("MIN_START_DELAY_MS", 0),
		MaxStartDelayMs:  parseEnvInt("MAX_START_DELAY_MS", 0),
		MinDelayMs:       parseEnvInt("MIN_DELAY_MS", 10),
		MaxDelayMs:       parseEnvInt("MAX_DELAY_MS", 100),
		BurnCPU:          parseEnvBool("BURN_CPU", false),
		CPUComplexity:    parseEnvInt("CPU_COMPLEXITY", 50000),
		ExternalServices: extSvc,
		MaxCallDepth:     parseEnvInt("MAX_CALL_DEPTH", 5),
		RequestTimeout:   parseEnvInt("REQUEST_TIMEOUT", 5),
		Hostname:         loadHostname(),
	}
}

func burnCPU(complexity int) time.Duration {
	start := time.Now()
	res := 0.0
	for i := 0; i < complexity; i++ {
		res += math.Sqrt(float64(i)) * math.Pow(float64(i), 0.5)
	}
	sink = res
	return time.Since(start)
}

func computeDelay(minMs, maxMs int) time.Duration {
	if minMs >= maxMs {
		return time.Duration(minMs) * time.Millisecond
	}
	delay := rand.Intn(maxMs-minMs) + minMs
	return time.Duration(delay) * time.Millisecond
}

func getDepth(r *http.Request) int {
	depthStr := r.Header.Get("X-Call-Depth")
	if depthStr == "" {
		return 0
	}
	d, err := strconv.Atoi(depthStr)
	if err != nil {
		return 0
	}
	return d
}

func getCaller(r *http.Request) string {
	callerStr := r.Header.Get("X-Caller-Hostname")
	if callerStr == "" {
		return "Externo"
	}
	return callerStr
}

func prioritizeStatusCode(statuses []int) int {
	worst := 200
	for _, s := range statuses {
		if s >= 500 {
			if worst < 500 || s > worst {
				worst = s
			}
		} else if s >= 400 {
			if worst < 400 || (worst >= 400 && worst < 500 && s > worst) {
				worst = s
			}
		} else {
			if worst < 400 && s > worst {
				worst = s
			}
		}
	}
	return worst
}

func main() {
	cfg := loadConfig()

	log.Printf("Starting Chaos Target on port %s", cfg.Port)
	log.Printf("Config loaded: MIN_START_DELAY_MS=%d MAX_START_DELAY_MS=%d MIN_DELAY_MS=%d MAX_DELAY_MS=%d BURN_CPU=%t CPU_COMPLEXITY=%d MAX_CALL_DEPTH=%d URLS=%v HOSTNAME=%s",
		cfg.MinStartDelayMs, cfg.MaxStartDelayMs, cfg.MinDelayMs, cfg.MaxDelayMs, cfg.BurnCPU, cfg.CPUComplexity, cfg.MaxCallDepth, cfg.ExternalServices, cfg.Hostname)

	if cfg.MinStartDelayMs+cfg.MaxStartDelayMs > 0 {
		startDelay := computeDelay(cfg.MinStartDelayMs, cfg.MaxStartDelayMs)
		log.Printf("Delay Start, Please wait %dms", startDelay.Milliseconds())
		time.Sleep(startDelay)
		log.Printf("Application Started")
	}

	rand.Seed(time.Now().UnixNano())

	httpClient := &http.Client{
		Timeout: time.Duration(cfg.RequestTimeout) * time.Second,
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		depth := getDepth(r)
		caller := getCaller(r)

		log.Printf("Request received: caller=%s, depth=%d", caller, depth)
		if depth >= cfg.MaxCallDepth {
			// Bypass
			if cfg.MaxCallDepth < 0 {
				log.Printf("Circuit Breaker Bypassed, MaxCallDepth=%d, CurrentDepth=%d", cfg.MaxCallDepth, depth)
			} else {
				log.Printf("Circuit Breaker Opened, MaxCallDepth=%d, CurrentDepth=%d", cfg.MaxCallDepth, depth)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(HealthResponse{
					StatusCode:   200,
					ModeActive:   "circuit_breaker_open",
					DepthLevel:   depth,
					ReachedLimit: true,
					TotalTime:    fmt.Sprintf("%v", time.Since(start)),
				})
				return
			}
		}

		var cpuDur time.Duration
		if cfg.BurnCPU {
			cpuDur = burnCPU(cfg.CPUComplexity)
		}

		isAggregator := len(cfg.ExternalServices) > 0
		var delayDur time.Duration
		finalStatus := 200
		servicesCalled := make(map[string]string)
		mode := "standalone"

		if isAggregator {
			mode = "service_chaining (BFF)"
			var wg sync.WaitGroup
			var mu sync.Mutex
			statusCodes := make([]int, 0, len(cfg.ExternalServices))

			for _, url := range cfg.ExternalServices {
				wg.Add(1)
				go func(targetURL string) {
					defer wg.Done()
					req, err := http.NewRequest("GET", targetURL, nil)
					if err != nil {
						mu.Lock()
						servicesCalled[targetURL] = "error (bad req)"
						statusCodes = append(statusCodes, 500)
						mu.Unlock()
						return
					}

					req.Header.Set("X-Call-Depth", strconv.Itoa(depth+1))
					req.Header.Set("X-Caller-Hostname", cfg.Hostname)

					resp, err := httpClient.Do(req)

					mu.Lock()
					defer mu.Unlock()

					if err != nil {
						servicesCalled[targetURL] = "error / timeout"
						statusCodes = append(statusCodes, 504)
					} else {
						defer resp.Body.Close()
						servicesCalled[targetURL] = strconv.Itoa(resp.StatusCode)
						statusCodes = append(statusCodes, resp.StatusCode)
					}
				}(url)
			}

			wg.Wait()
			delayDur = time.Since(start) - cpuDur
			finalStatus = prioritizeStatusCode(statusCodes)

		} else {
			delayDur = computeDelay(cfg.MinDelayMs, cfg.MaxDelayMs)
			if delayDur > 0 {
				time.Sleep(delayDur)
			}
		}

		totalDur := time.Since(start)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(finalStatus)

		log.Printf("Request ended: caller=%s, status=%d, total time=%v", caller, finalStatus, totalDur)
		resp := HealthResponse{
			StatusCode:     finalStatus,
			ModeActive:     mode,
			DepthLevel:     depth,
			ReachedLimit:   false,
			CPUTime:        fmt.Sprintf("%v", cpuDur),
			WaitTime:       fmt.Sprintf("%v", delayDur),
			TotalTime:      fmt.Sprintf("%v", totalDur),
			ServicesCalled: servicesCalled,
		}

		json.NewEncoder(w).Encode(resp)
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	http.HandleFunc("/status/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Request received to /status/%s", r.URL.Path)
		statusCodeStr := strings.TrimPrefix(r.URL.Path, "/status/")
		statusCodeInt, err := strconv.Atoi(statusCodeStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCodeInt)
		json.NewEncoder(w).Encode(map[string]string{"status": statusCodeStr})
	})

	if err := http.ListenAndServe(":"+cfg.Port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
