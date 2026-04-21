package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// TDataPoint is a single row from an Apex tdata schedule.
// T is seconds since midnight. Ch holds the 13 channel values (0–100).
type TDataPoint struct {
	T  int   `json:"t"`
	Ch []int `json:"ch"`
}

// OutputProgram is the API representation of a single output's Apex program.
type OutputProgram struct {
	DID   string       `json:"did"`
	Name  string       `json:"name"`
	Type  string       `json:"type"`
	Icon  string       `json:"icon"`
	Prog  string       `json:"prog"`
	TData []TDataPoint `json:"tdata,omitempty"`
}

// programCache holds a fetched program list with a simple TTL.
type programCache struct {
	mu       sync.Mutex
	programs []OutputProgram
	fetchedAt time.Time
	ttl      time.Duration
}

func newProgramCache(ttl time.Duration) *programCache {
	return &programCache{ttl: ttl}
}

func (c *programCache) get() ([]OutputProgram, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.programs == nil || time.Since(c.fetchedAt) > c.ttl {
		return nil, false
	}
	return c.programs, true
}

func (c *programCache) set(programs []OutputProgram) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.programs = programs
	c.fetchedAt = time.Now()
}

// parseTData extracts the first tdata block from an Apex program string.
// The program may contain duplicate tdata blocks (normal vs feed-mode); only
// the first block (up to the second "Fallback" line) is returned.
func parseTData(prog string) []TDataPoint {
	var points []TDataPoint
	seenTData := false

	for _, raw := range strings.Split(prog, "\n") {
		line := strings.TrimSpace(raw)

		if strings.HasPrefix(line, "tdata ") {
			seenTData = true
			rest := strings.TrimPrefix(line, "tdata ")
			parts := strings.SplitN(rest, ",", 14)
			if len(parts) < 1 {
				continue
			}

			// Parse HH:MM:SS into seconds since midnight.
			secs, err := parseHMS(parts[0])
			if err != nil {
				continue
			}

			ch := make([]int, 13)
			for i := 1; i < len(parts) && i <= 13; i++ {
				v, _ := strconv.Atoi(strings.TrimSpace(parts[i]))
				ch[i-1] = v
			}

			points = append(points, TDataPoint{T: secs, Ch: ch})
			continue
		}

		// Stop at the second Fallback line (feed-mode duplicate block).
		if seenTData && strings.HasPrefix(line, "Fallback") {
			break
		}
	}

	return points
}

func parseHMS(s string) (int, error) {
	parts := strings.Split(strings.TrimSpace(s), ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid time %q", s)
	}
	h, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, err
	}
	m, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, err
	}
	sec, err := strconv.Atoi(parts[2])
	if err != nil {
		return 0, err
	}
	return h*3600 + m*60 + sec, nil
}

// HandleProgramList fetches Apex output programs from /rest/config, caches the
// result for 5 minutes, parses tdata blocks for variable outputs, and returns
// the full list as JSON.
func (s *Server) HandleProgramList(w http.ResponseWriter, r *http.Request) {
	if programs, ok := s.programCache.get(); ok {
		writeJSON(w, http.StatusOK, map[string]any{"programs": programs})
		return
	}

	cfg, err := s.apex.Config(r.Context())
	if err != nil {
		s.logger.Error("fetching apex config", "err", err)
		writeError(w, http.StatusBadGateway, "failed to fetch programs from Apex", "apex_error")
		return
	}

	programs := make([]OutputProgram, 0, len(cfg.OConf))
	for _, oc := range cfg.OConf {
		p := OutputProgram{
			DID:  oc.DID,
			Name: oc.Name,
			Type: oc.Type,
			Icon: oc.Icon,
			Prog: strings.TrimSpace(oc.Prog),
		}
		if strings.Contains(oc.Prog, "\ntdata ") || strings.HasPrefix(oc.Prog, "tdata ") {
			p.TData = parseTData(oc.Prog)
		}
		programs = append(programs, p)
	}

	s.programCache.set(programs)
	writeJSON(w, http.StatusOK, map[string]any{"programs": programs})
}
