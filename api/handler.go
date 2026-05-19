package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/loafoe/go-occp-server/ocpp"
)

// Handler provides HTTP API handlers for controlling charge points
type Handler struct {
	server *ocpp.Server
}

// NewHandler creates a new API handler
func NewHandler(server *ocpp.Server) *Handler {
	return &Handler{server: server}
}

// ChargePointInfo represents charge point information for API responses
type ChargePointInfo struct {
	ID              string            `json:"id"`
	Vendor          string            `json:"vendor"`
	Model           string            `json:"model"`
	SerialNumber    string            `json:"serialNumber"`
	FirmwareVersion string            `json:"firmwareVersion"`
	Status          string            `json:"status"`
	ConnectorStatus map[int]string    `json:"connectorStatus"`
	LastHeartbeat   string            `json:"lastHeartbeat"`
	Transactions    []TransactionInfo `json:"transactions,omitempty"`
}

// TransactionInfo represents transaction information for API responses
type TransactionInfo struct {
	ID          int    `json:"id"`
	ConnectorID int    `json:"connectorId"`
	IdTag       string `json:"idTag"`
	MeterStart  int    `json:"meterStart"`
	MeterStop   int    `json:"meterStop,omitempty"`
	StartTime   string `json:"startTime"`
	StopTime    string `json:"stopTime,omitempty"`
	Active      bool   `json:"active"`
	EnergyWh    int    `json:"energyWh,omitempty"`
}

// MeterValueInfo represents meter values for API responses
type MeterValueInfo struct {
	Timestamp string         `json:"timestamp"`
	Values    map[string]any `json:"values"`
}

// RegisterRoutes registers all API routes
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/chargepoints", h.handleChargePoints)
	mux.HandleFunc("/api/chargepoints/", h.handleChargePoint)
}

func (h *Handler) handleChargePoints(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cps := h.server.GetAllChargePoints()
	result := make([]ChargePointInfo, 0, len(cps))

	for _, cp := range cps {
		info := h.chargePointToInfo(cp)
		result = append(result, info)
	}

	h.writeJSON(w, result)
}

func (h *Handler) handleChargePoint(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[len("/api/chargepoints/"):]
	parts := splitPath(path)

	if len(parts) == 0 {
		http.Error(w, "Charge point ID required", http.StatusBadRequest)
		return
	}

	cpID := parts[0]
	cp, ok := h.server.GetChargePoint(cpID)
	if !ok {
		http.Error(w, "Charge point not found", http.StatusNotFound)
		return
	}

	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.writeJSON(w, h.chargePointToInfo(cp))
		return
	}

	action := parts[1]
	switch action {
	case "unlock":
		h.handleUnlock(w, r, cpID)
	case "lock":
		h.handleLock(w, r, cpID)
	case "status":
		h.handleStatus(w, r, cp)
	case "meter":
		h.handleMeterValues(w, r, cp, parts)
	case "start":
		h.handleRemoteStart(w, r, cpID)
	case "stop":
		h.handleRemoteStop(w, r, cpID)
	case "config":
		h.handleConfig(w, r, cpID)
	default:
		http.Error(w, "Unknown action", http.StatusBadRequest)
	}
}

func (h *Handler) handleUnlock(w http.ResponseWriter, r *http.Request, cpID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	connectorID := 1
	if cid := r.URL.Query().Get("connector"); cid != "" {
		if id, err := strconv.Atoi(cid); err == nil {
			connectorID = id
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	status, err := h.server.UnlockConnector(ctx, cpID, connectorID)
	if err != nil {
		h.writeError(w, err)
		return
	}

	h.writeJSON(w, map[string]string{"status": status})
}

func (h *Handler) handleLock(w http.ResponseWriter, r *http.Request, cpID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	connectorID := 1
	if cid := r.URL.Query().Get("connector"); cid != "" {
		if id, err := strconv.Atoi(cid); err == nil {
			connectorID = id
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	status, err := h.server.ChangeAvailability(ctx, cpID, connectorID, ocpp.AvailabilityTypeInoperative)
	if err != nil {
		h.writeError(w, err)
		return
	}

	h.writeJSON(w, map[string]string{"status": status})
}

func (h *Handler) handleStatus(w http.ResponseWriter, r *http.Request, cp *ocpp.ChargePoint) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.writeJSON(w, map[string]any{
		"id":              cp.ID,
		"status":          cp.Status,
		"connectorStatus": cp.ConnectorStatus,
		"lastHeartbeat":   cp.LastHeartbeat.Format(time.RFC3339),
	})
}

func (h *Handler) handleMeterValues(w http.ResponseWriter, r *http.Request, cp *ocpp.ChargePoint, parts []string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	connectorID := 1
	if len(parts) > 2 {
		if id, err := strconv.Atoi(parts[2]); err == nil {
			connectorID = id
		}
	}

	meterValues := cp.MeterValues[connectorID]
	result := make([]MeterValueInfo, 0, len(meterValues))

	for _, mv := range meterValues {
		info := MeterValueInfo{
			Timestamp: mv.Timestamp,
			Values:    make(map[string]any),
		}
		for _, sv := range mv.SampledValue {
			key := sv.Measurand
			if key == "" {
				key = "Energy.Active.Import.Register"
			}
			info.Values[key] = map[string]string{
				"value": sv.Value,
				"unit":  sv.Unit,
			}
		}
		result = append(result, info)
	}

	h.writeJSON(w, result)
}

func (h *Handler) handleRemoteStart(w http.ResponseWriter, r *http.Request, cpID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	connectorID := 1
	if cid := r.URL.Query().Get("connector"); cid != "" {
		if id, err := strconv.Atoi(cid); err == nil {
			connectorID = id
		}
	}

	idTag := r.URL.Query().Get("tag")
	if idTag == "" {
		idTag = "REMOTE"
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	status, err := h.server.RemoteStartTransaction(ctx, cpID, connectorID, idTag)
	if err != nil {
		h.writeError(w, err)
		return
	}

	h.writeJSON(w, map[string]string{"status": status})
}

func (h *Handler) handleRemoteStop(w http.ResponseWriter, r *http.Request, cpID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	txIDStr := r.URL.Query().Get("transaction")
	if txIDStr == "" {
		http.Error(w, "Transaction ID required", http.StatusBadRequest)
		return
	}

	txID, err := strconv.Atoi(txIDStr)
	if err != nil {
		http.Error(w, "Invalid transaction ID", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	status, err := h.server.RemoteStopTransaction(ctx, cpID, txID)
	if err != nil {
		h.writeError(w, err)
		return
	}

	h.writeJSON(w, map[string]string{"status": status})
}

func (h *Handler) handleConfig(w http.ResponseWriter, r *http.Request, cpID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var keys []string
	if k := r.URL.Query().Get("keys"); k != "" {
		keys = splitPath(k)
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	config, err := h.server.GetConfiguration(ctx, cpID, keys)
	if err != nil {
		h.writeError(w, err)
		return
	}

	h.writeJSON(w, config)
}

func (h *Handler) chargePointToInfo(cp *ocpp.ChargePoint) ChargePointInfo {
	info := ChargePointInfo{
		ID:              cp.ID,
		Vendor:          cp.Vendor,
		Model:           cp.Model,
		SerialNumber:    cp.SerialNumber,
		FirmwareVersion: cp.FirmwareVersion,
		Status:          cp.Status,
		ConnectorStatus: cp.ConnectorStatus,
		LastHeartbeat:   cp.LastHeartbeat.Format(time.RFC3339),
		Transactions:    make([]TransactionInfo, 0),
	}

	for _, tx := range cp.Transactions {
		txInfo := TransactionInfo{
			ID:          tx.ID,
			ConnectorID: tx.ConnectorID,
			IdTag:       tx.IdTag,
			MeterStart:  tx.MeterStart,
			MeterStop:   tx.MeterStop,
			StartTime:   tx.StartTime.Format(time.RFC3339),
			Active:      tx.Active,
		}
		if !tx.StopTime.IsZero() {
			txInfo.StopTime = tx.StopTime.Format(time.RFC3339)
		}
		if tx.MeterStop > tx.MeterStart {
			txInfo.EnergyWh = tx.MeterStop - tx.MeterStart
		}
		info.Transactions = append(info.Transactions, txInfo)
	}

	return info
}

func (h *Handler) writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(data)
}

func (h *Handler) writeError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func splitPath(path string) []string {
	var parts []string
	for _, p := range split(path, '/') {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

func split(s string, sep rune) []string {
	var parts []string
	var current string
	for _, c := range s {
		if c == sep {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	parts = append(parts, current)
	return parts
}
