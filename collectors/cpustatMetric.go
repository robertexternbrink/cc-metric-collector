// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
// additional authors:
// Holger Obermaier (NHR@KIT)

package collectors

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	cclog "github.com/ClusterCockpit/cc-lib/v2/ccLogger"
	lp "github.com/ClusterCockpit/cc-lib/v2/ccMessage"
	sysconf "github.com/tklauser/go-sysconf"
)

const CPUSTATFILE = `/proc/stat`

var cpustatInputFile = CPUSTATFILE
var cpustatOpenFile = os.Open

type CpustatCollectorConfig struct {
	ExcludeMetrics []string `json:"exclude_metrics,omitempty"`
	excludeNumCPUs bool
}

type CpustatCollector struct {
	metricCollector

	config        CpustatCollectorConfig
	lastTimestamp time.Time // Store time stamp of last tick to derive values
	matches       map[string]int
	cputags       map[string]map[string]string
	nodetags      map[string]string
	olddata       map[string]map[string]int64
}

func readCPUStatLines(file *os.File) ([][]string, error) {
	var lines [][]string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 0 || !strings.HasPrefix(fields[0], "cpu") {
			continue
		}
		if fields[0] != "cpu" {
			if _, err := strconv.Atoi(strings.TrimPrefix(fields[0], "cpu")); err != nil {
				return nil, fmt.Errorf("invalid CPU identifier %q", fields[0])
			}
		}
		if len(fields) < 5 {
			return nil, fmt.Errorf("incomplete CPU counters for %s", fields[0])
		}
		for index := 1; index < len(fields) && index <= 10; index++ {
			if _, err := strconv.ParseInt(fields[index], 0, 64); err != nil {
				return nil, fmt.Errorf("invalid CPU counter for %s: %w", fields[0], err)
			}
		}
		lines = append(lines, fields)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return nil, fmt.Errorf("no CPU counters found")
	}
	return lines, nil
}

func (m *CpustatCollector) Init(config json.RawMessage) error {
	m.name = "CpustatCollector"
	if err := m.setup(); err != nil {
		return fmt.Errorf("%s Init(): setup() call failed: %s", m.name, err.Error())
	}
	m.parallel = true
	m.meta = map[string]string{
		"source": m.name,
		"group":  "CPU",
	}
	m.nodetags = map[string]string{
		"type": "node",
	}
	if len(config) > 0 {
		d := json.NewDecoder(bytes.NewReader(config))
		d.DisallowUnknownFields()
		if err := d.Decode(&m.config); err != nil {
			return fmt.Errorf("%s Init(): Error decoding JSON config: %s", m.name, err.Error())
		}
	}
	matches := map[string]int{
		"cpu_user":       1,
		"cpu_nice":       2,
		"cpu_system":     3,
		"cpu_idle":       4,
		"cpu_iowait":     5,
		"cpu_irq":        6,
		"cpu_softirq":    7,
		"cpu_steal":      8,
		"cpu_guest":      9,
		"cpu_guest_nice": 10,
	}

	m.matches = make(map[string]int)
	for match, index := range matches {
		m.matches[match] = index
	}
	m.config.excludeNumCPUs = slices.Contains(m.config.ExcludeMetrics, "num_cpus")

	// Check input file
	file, err := cpustatOpenFile(cpustatInputFile)
	if err != nil {
		return fmt.Errorf("%s Init(): Failed to open file '%s': %s", m.name, cpustatInputFile, err.Error())
	}

	// Pre-generate tags for all CPUs
	m.cputags = make(map[string]map[string]string)
	m.olddata = make(map[string]map[string]int64)
	lines, err := readCPUStatLines(file)
	if err != nil {
		if closeErr := file.Close(); closeErr != nil {
			cclog.ComponentErrorf(m.name, "Init(): Failed to close file '%s': %v", cpustatInputFile, closeErr)
		}
		return fmt.Errorf("%s Init(): Failed to read CPU counters: %w", m.name, err)
	}
	for _, linefields := range lines {
		if strings.Compare(linefields[0], "cpu") == 0 {
			// Kernel system statistics for all CPUs
			m.olddata["cpu"] = make(map[string]int64)
			for k, v := range m.matches {
				if v < len(linefields) {
					m.olddata["cpu"][k], _ = strconv.ParseInt(linefields[v], 0, 64)
				}
			}
		} else if strings.HasPrefix(linefields[0], "cpu") && strings.Compare(linefields[0], "cpu") != 0 {
			// Kernel system statistics per CPU
			cpustr := strings.TrimLeft(linefields[0], "cpu")
			cpu, _ := strconv.Atoi(cpustr)
			m.cputags[linefields[0]] = map[string]string{
				"type":    "hwthread",
				"type-id": strconv.Itoa(cpu),
			}
			m.olddata[linefields[0]] = make(map[string]int64)
			for k, v := range m.matches {
				if v < len(linefields) {
					m.olddata[linefields[0]][k], _ = strconv.ParseInt(linefields[v], 0, 64)
				}
			}
		}
	}

	// Close file
	if err := file.Close(); err != nil {
		return fmt.Errorf("%s Init(): Failed to close file '%s': %s", m.name, cpustatInputFile, err.Error())
	}

	m.lastTimestamp = time.Now()
	m.init = true
	return nil
}

func (m *CpustatCollector) parseStatLine(linefields []string, tags map[string]string, output chan lp.CCMessage, now time.Time, tsdelta time.Duration) {
	if len(linefields) == 0 {
		return
	}
	if m.olddata[linefields[0]] == nil {
		m.olddata[linefields[0]] = make(map[string]int64)
	}
	values := make(map[string]float64)
	clktck, _ := sysconf.Sysconf(sysconf.SC_CLK_TCK)
	for match, index := range m.matches {
		if len(match) > 0 {
			if index >= len(linefields) {
				delete(m.olddata[linefields[0]], match)
				continue
			}
			x, err := strconv.ParseInt(linefields[index], 0, 64)
			if err == nil {
				previous, exists := m.olddata[linefields[0]][match]
				m.olddata[linefields[0]][match] = x // Store new value for next run
				if !exists {
					continue
				}
				vdiff := x - previous
				values[match] = float64(vdiff) / float64(tsdelta.Seconds()) / float64(clktck)
			} else {
				delete(m.olddata[linefields[0]], match)
			}
		}
	}

	sum := float64(0)
	for name, value := range values {
		sum += value
		if slices.Contains(m.config.ExcludeMetrics, name) {
			continue
		}
		y, err := lp.NewMetric(name, tags, m.meta, value*100, now)
		if err == nil {
			y.AddTag("unit", "Percent")
			output <- y
		}
	}
	if v, ok := values["cpu_idle"]; ok && !slices.Contains(m.config.ExcludeMetrics, "cpu_used") {
		sum -= v
		y, err := lp.NewMetric("cpu_used", tags, m.meta, sum*100, now)
		if err == nil {
			y.AddTag("unit", "Percent")
			output <- y
		}
	}
}

func (m *CpustatCollector) Read(interval time.Duration, output chan lp.CCMessage) {
	if !m.init {
		return
	}
	num_cpus := 0
	now := time.Now()
	tsdelta := now.Sub(m.lastTimestamp)

	file, err := cpustatOpenFile(cpustatInputFile)
	if err != nil {
		cclog.ComponentError(
			m.name,
			fmt.Sprintf("Read(): Failed to open file '%s': %v", cpustatInputFile, err.Error()))
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			cclog.ComponentError(
				m.name,
				fmt.Sprintf("Read(): Failed to close file '%s': %v", string(cpustatInputFile), err.Error()))
		}
	}()

	lines, err := readCPUStatLines(file)
	if err != nil {
		cclog.ComponentErrorf(m.name, "Read(): Failed to read CPU counters: %v", err)
		return
	}
	seen := make(map[string]bool, len(lines))
	for _, linefields := range lines {
		seen[linefields[0]] = true
		if strings.Compare(linefields[0], "cpu") == 0 {
			m.parseStatLine(linefields, m.nodetags, output, now, tsdelta)
		} else if strings.HasPrefix(linefields[0], "cpu") {
			if m.cputags[linefields[0]] == nil {
				cpu, _ := strconv.Atoi(strings.TrimPrefix(linefields[0], "cpu"))
				m.cputags[linefields[0]] = map[string]string{
					"type":    "hwthread",
					"type-id": strconv.Itoa(cpu),
				}
			}
			m.parseStatLine(linefields, m.cputags[linefields[0]], output, now, tsdelta)
			num_cpus++
		}
	}
	for cpu := range m.olddata {
		if !seen[cpu] {
			delete(m.olddata, cpu)
			delete(m.cputags, cpu)
		}
	}

	if !m.config.excludeNumCPUs {
		if num_cpus_metric, err := lp.NewMetric("num_cpus", m.nodetags, m.meta, num_cpus, now); err == nil {
			output <- num_cpus_metric
		}
	}

	m.lastTimestamp = now
}

func (m *CpustatCollector) Close() {
	m.init = false
}
