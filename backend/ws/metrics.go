package ws

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// SystemMetrics holds host-level system statistics.
type SystemMetrics struct {
	CPU struct {
		Percent float64 `json:"percent"` // overall CPU usage %
		Cores   int     `json:"cores"`
	} `json:"cpu"`
	Memory struct {
		UsedMB      float64 `json:"used_mb"`
		TotalMB     float64 `json:"total_mb"`
		UsedPercent float64 `json:"used_percent"`
	} `json:"memory"`
	Network struct {
		RxBytes int64 `json:"rx_bytes"`
		TxBytes int64 `json:"tx_bytes"`
	} `json:"network"`
	Wifi struct {
		Connected bool   `json:"connected"`
		SSID      string `json:"ssid,omitempty"`
	} `json:"wifi"`
	Battery struct {
		Percent   int  `json:"percent"` // 0-100
		Charging  bool `json:"charging"`
		Available bool `json:"available"`
	} `json:"battery"`
	Volume struct {
		Percent int  `json:"percent"` // 0-100, 0=muted
		Muted   bool `json:"muted"`
	} `json:"volume"`
	Temperature struct {
		Celsius float64 `json:"celsius"`
		Source  string  `json:"source"` // e.g. "cpu", "acpitz"
	} `json:"temperature"`
	Audio struct {
		Active   bool   `json:"active"`
		Source   string `json:"source,omitempty"` // e.g. "fun-audio-chat", "none"
		Icon     string `json:"icon"`              // "🔊" or "🔇"
	} `json:"audio"`
	Timestamp int64 `json:"timestamp"`
}

// metricsCollector aggregates system metrics with caching.
type metricsCollector struct {
	mu         sync.RWMutex
	cached     *SystemMetrics
	lastFetch  time.Time
	cacheTTL   time.Duration // minimum interval between fetches
	audioActive bool
	audioSource string
}

func newMetricsCollector() *metricsCollector {
	return &metricsCollector{
		cacheTTL: 2 * time.Second,
	}
}

var globalCollector = newMetricsCollector()

// FetchMetrics returns cached metrics if fresh, otherwise refreshes.
func (mc *metricsCollector) FetchMetrics() *SystemMetrics {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	now := time.Now()
	if mc.cached != nil && now.Sub(mc.lastFetch) < mc.cacheTTL {
		return mc.cached
	}
	mc.cached = mc.gather()
	mc.lastFetch = now
	return mc.cached
}

func (mc *metricsCollector) SetAudioState(active bool, source string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.audioActive = active
	mc.audioSource = source
}

func (mc *metricsCollector) gather() *SystemMetrics {
	m := &SystemMetrics{}
	m.Timestamp = time.Now().UnixMilli()

	// CPU — read /proc/stat
	cpuPct, cores := readCPUUsage()
	m.CPU.Percent = cpuPct
	m.CPU.Cores = cores

	// Memory — read /proc/meminfo
	usedMB, totalMB := readMemInfo()
	m.Memory.UsedMB = usedMB
	m.Memory.TotalMB = totalMB
	if totalMB > 0 {
		m.Memory.UsedPercent = (usedMB / totalMB) * 100
	}

	// Network I/O — read /proc/net/dev (first non-loopback interface)
	rx, tx := readNetworkIO()
	m.Network.RxBytes = rx
	m.Network.TxBytes = tx

	// Wifi — check /sys/class/net/<iface>/operstate
	m.Wifi.Connected, m.Wifi.SSID = readWifiStatus()

	// Battery — read /sys/class/power_supply/
	m.Battery.Percent, m.Battery.Charging, m.Battery.Available = readBatteryStatus()

	// Volume — check pulse audio or /proc/self/fd
	m.Volume.Percent, m.Volume.Muted = readVolumeStatus()

	// Temperature — try multiple sources
	celsius, source := readTemperature()
	m.Temperature.Celsius = celsius
	m.Temperature.Source = source

	// Audio state
	m.Audio.Active = mc.audioActive
	m.Audio.Source = mc.audioSource
	if m.Audio.Source == "" {
		m.Audio.Source = "none"
	}
	if m.Audio.Active && m.Audio.Source != "none" {
		m.Audio.Icon = "🔊"
	} else {
		m.Audio.Icon = "🔇"
	}

	return m
}

// readCPUUsage reads /proc/stat and returns overall CPU usage % and number of cores.
// Uses a single snapshot to approximate CPU usage (idle/total ratio).
func readCPUUsage() (float64, int) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, 0
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		var total, idle uint64
		for i := 1; i < len(fields); i++ {
			v, _ := strconv.ParseUint(fields[i], 10, 64)
			total += v
			if i == 4 { // idle
				idle = v
			}
		}
		if total > 0 {
			// Return usage % = (1 - idle/total) * 100
			usagePct := (1.0 - float64(idle)/float64(total)) * 100
			return usagePct, runtime.NumCPU()
		}
	}
	return 0, runtime.NumCPU()
}

// readMemInfo reads /proc/meminfo and returns used and total MB.
func readMemInfo() (float64, float64) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0
	}
	var memTotal, memFree, buffers, cached int64
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "MemTotal:":
			memTotal, _ = strconv.ParseInt(fields[1], 10, 64)
		case "MemFree:":
			memFree, _ = strconv.ParseInt(fields[1], 10, 64)
		case "Buffers:":
			buffers, _ = strconv.ParseInt(fields[1], 10, 64)
		case "Cached:":
			cached, _ = strconv.ParseInt(fields[1], 10, 64)
		}
	}
	totalMB := float64(memTotal) / 1024
	usedMB := float64(memTotal-memFree-buffers-cached) / 1024
	return usedMB, totalMB
}

// readNetworkIO returns total rx + tx bytes from /proc/net/dev.
func readNetworkIO() (rx, tx int64) {
	data, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return 0, 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "  lo:") || strings.HasPrefix(line, "lo:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 10 {
			continue
		}
		// skip interface name field (ends with :)
		rxB, _ := strconv.ParseInt(fields[1], 10, 64)
		txB, _ := strconv.ParseInt(fields[9], 10, 64)
		rx += rxB
		tx += txB
	}
	return rx, tx
}

// readTemperature tries several temperature sources and returns Celsius.
func readTemperature() (float64, string) {
	sources := []string{
		"/sys/class/thermal/thermal_zone0/temp",
		"/sys/class/hwmon/hwmon0/temp1_input",
		"/sys/class/hwmon/hwmon1/temp1_input",
		"/sys/class/thermal/thermal_zone1/temp",
	}
	for _, path := range sources {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		val := strings.TrimSpace(string(data))
		milliCelsius, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			continue
		}
		return float64(milliCelsius) / 1000.0, path
	}
	return 0, "unavailable"
}

// readWifiStatus checks wifi interface state and returns connected bool + SSID string.
// Matches both legacy "wlan" and modern "wl*" interface naming (e.g. wlp2s0).
func readWifiStatus() (bool, string) {
	// Check for wifi interfaces via /sys/class/net/
	entries, err := os.ReadDir("/sys/class/net")
	if err != nil {
		return false, ""
	}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, "wlan") && !strings.HasPrefix(name, "wl") {
			continue
		}
		operstate, err := os.ReadFile("/sys/class/net/" + name + "/operstate")
		if err != nil {
			continue
		}
		connected := strings.TrimSpace(string(operstate)) == "up"
		var ssid string
		if connected {
			// Try reading /proc/net/wireless for link quality (indicates active connection)
			if wlData, err := os.ReadFile("/proc/net/wireless"); err == nil {
				for _, wlLine := range strings.Split(string(wlData), "\n") {
					wlFields := strings.Fields(wlLine)
					if len(wlFields) >= 2 && wlFields[0] == name+":" {
						// Interface found in wireless table — we know it's a wifi connection
						ssid = name
						break
					}
				}
			}
			// If /proc/net/wireless didn't yield a match, use interface name as label
			if ssid == "" {
				ssid = name
			}
		}
		return connected, ssid
	}
	return false, ""
}

// readBatteryStatus reads battery status from /sys/class/power_supply/.
func readBatteryStatus() (percent int, charging bool, available bool) {
	entries, err := os.ReadDir("/sys/class/power_supply")
	if err != nil {
		return 0, false, false
	}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, "BAT") {
			continue
		}
		// Read capacity
		capData, err := os.ReadFile("/sys/class/power_supply/" + name + "/capacity")
		if err == nil {
			pct, _ := strconv.Atoi(strings.TrimSpace(string(capData)))
			percent = pct
		}
		// Read status
		statusData, err := os.ReadFile("/sys/class/power_supply/" + name + "/status")
		if err == nil {
			status := strings.TrimSpace(string(statusData))
			charging = status == "Charging" || status == "Full"
		}
		return percent, charging, true
	}
	return 0, false, false
}

// readVolumeStatus attempts to read system volume via PipeWire, PulseAudio, or ALSA.
// Returns (volume_percent 0-100, muted). Falls back to (100, false) on failure.
func readVolumeStatus() (int, bool) {
	// 1. Try PipeWire (wpctl)
	if out, err := exec.Command("wpctl", "get-volume").Output(); err == nil {
		// Output formats: "Volume: 0.85  [MUTED]" or "Volume: 0.75"
		fields := strings.Fields(strings.TrimSpace(string(out)))
		if len(fields) >= 2 {
			if v, err := strconv.ParseFloat(fields[1], 64); err == nil {
				pct := int(v * 100)
				if pct > 100 {
					pct = 100
				} else if pct < 0 {
					pct = 0
				}
				return pct, strings.Contains(string(out), "MUTED")
			}
		}
	}

	// 2. Try PulseAudio (pactl)
	if out, err := exec.Command("pactl", "list", "sinks").Output(); err == nil {
		output := string(out)
		// Look for "Mute: yes" or "Mute: no"
		muted := strings.Contains(output, "Mute: yes")
		// Look for "Volume: front-left: NNNN / N%"
		for _, line := range strings.Split(output, "\n") {
			if strings.Contains(line, "front-left:") || strings.Contains(line, "Volume:") {
				fields := strings.Fields(line)
				for _, f := range fields {
					if strings.HasSuffix(f, "%") {
						val := strings.TrimSuffix(f, "%")
						if v, err := strconv.Atoi(val); err == nil {
							if v > 100 {
								v = 100
							} else if v < 0 {
								v = 0
							}
							return v, muted
						}
					}
				}
			}
		}
	}

	// 3. Try ALSA via /proc/asound/card*/pcm*/sub*/status
	entries, err := os.ReadDir("/proc/asound")
	if err == nil {
		for _, entry := range entries {
			if !strings.HasPrefix(entry.Name(), "card") {
				continue
			}
			cardDir := "/proc/asound/" + entry.Name()
			pcmEntries, err := os.ReadDir(cardDir)
			if err != nil {
				continue
			}
			for _, pcm := range pcmEntries {
				subDir := cardDir + "/" + pcm.Name() + "/sub0"
				statusFile := subDir + "/status"
				data, err := os.ReadFile(statusFile)
				if err != nil {
					continue
				}
				status := strings.TrimSpace(string(data))
				if strings.Contains(status, "RUNNING") || strings.Contains(status, "OPEN") {
					return 100, false
				}
			}
		}
	}

	// 4. Fallback: sensible defaults
	return 100, false
}

// ServeMetricsHTTP is the HTTP handler for GET /api/system/metrics.
func ServeMetricsHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	metrics := globalCollector.FetchMetrics()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

