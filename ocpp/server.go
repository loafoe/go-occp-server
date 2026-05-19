package ocpp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// ChargePoint represents a connected charge point
type ChargePoint struct {
	ID              string
	Vendor          string
	Model           string
	SerialNumber    string
	FirmwareVersion string
	Status          string
	ConnectorStatus map[int]string
	LastHeartbeat   time.Time
	MeterValues     map[int][]MeterValue
	Transactions    map[int]*Transaction
	conn            *websocket.Conn
	mu              sync.RWMutex
	pendingCalls    map[string]chan *CallResult
	pendingCallsMu  sync.Mutex
}

// Transaction represents an ongoing charging transaction
type Transaction struct {
	ID          int
	ConnectorID int
	IdTag       string
	MeterStart  int
	MeterStop   int
	StartTime   time.Time
	StopTime    time.Time
	Active      bool
}

// Server represents the OCPP Central System
type Server struct {
	chargePoints map[string]*ChargePoint
	mu           sync.RWMutex
	upgrader     websocket.Upgrader
	nextTxID     int
	txMu         sync.Mutex
}

// NewServer creates a new OCPP server
func NewServer() *Server {
	return &Server{
		chargePoints: make(map[string]*ChargePoint),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
			Subprotocols:    []string{"ocpp1.6"},
		},
		nextTxID: 1,
	}
}

// GetChargePoint returns a charge point by ID
func (s *Server) GetChargePoint(id string) (*ChargePoint, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp, ok := s.chargePoints[id]
	return cp, ok
}

// GetAllChargePoints returns all connected charge points
func (s *Server) GetAllChargePoints() []*ChargePoint {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cps := make([]*ChargePoint, 0, len(s.chargePoints))
	for _, cp := range s.chargePoints {
		cps = append(cps, cp)
	}
	return cps
}

// HandleWebSocket handles incoming WebSocket connections
func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/ocpp/")
	chargePointID := strings.TrimSuffix(path, "/")
	if chargePointID == "" {
		http.Error(w, "charge point ID required", http.StatusBadRequest)
		return
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	cp := &ChargePoint{
		ID:              chargePointID,
		conn:            conn,
		ConnectorStatus: make(map[int]string),
		MeterValues:     make(map[int][]MeterValue),
		Transactions:    make(map[int]*Transaction),
		pendingCalls:    make(map[string]chan *CallResult),
	}

	s.mu.Lock()
	s.chargePoints[chargePointID] = cp
	s.mu.Unlock()

	log.Printf("Charge point connected: %s", chargePointID)

	go s.handleConnection(cp)
}

func (s *Server) handleConnection(cp *ChargePoint) {
	defer func() {
		_ = cp.conn.Close()
		s.mu.Lock()
		delete(s.chargePoints, cp.ID)
		s.mu.Unlock()
		log.Printf("Charge point disconnected: %s", cp.ID)
	}()

	for {
		_, message, err := cp.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			return
		}

		log.Printf("Received from %s: %s", cp.ID, string(message))

		parsed, err := ParseMessage(message)
		if err != nil {
			log.Printf("Failed to parse message: %v", err)
			continue
		}

		switch msg := parsed.(type) {
		case *Call:
			s.handleCall(cp, msg)
		case *CallResult:
			s.handleCallResult(cp, msg)
		case *CallError:
			log.Printf("Received CallError from %s: %s - %s", cp.ID, msg.ErrorCode, msg.ErrorDescription)
		}
	}
}

func (s *Server) handleCall(cp *ChargePoint, call *Call) {
	var response []byte
	var err error

	switch call.Action {
	case ActionBootNotification:
		response, err = s.handleBootNotification(cp, call)
	case ActionHeartbeat:
		response, err = s.handleHeartbeat(cp, call)
	case ActionStatusNotification:
		response, err = s.handleStatusNotification(cp, call)
	case ActionMeterValues:
		response, err = s.handleMeterValues(cp, call)
	case ActionStartTransaction:
		response, err = s.handleStartTransaction(cp, call)
	case ActionStopTransaction:
		response, err = s.handleStopTransaction(cp, call)
	case ActionAuthorize:
		response, err = s.handleAuthorize(cp, call)
	case ActionDataTransfer:
		response, err = s.handleDataTransfer(cp, call)
	default:
		log.Printf("Unhandled action: %s", call.Action)
		response, err = CreateCallError(call.UniqueID, "NotImplemented", "Action not implemented", nil)
	}

	if err != nil {
		log.Printf("Failed to create response: %v", err)
		return
	}

	log.Printf("Sending to %s: %s", cp.ID, string(response))
	cp.mu.Lock()
	err = cp.conn.WriteMessage(websocket.TextMessage, response)
	cp.mu.Unlock()
	if err != nil {
		log.Printf("Failed to send response: %v", err)
	}
}

func (s *Server) handleCallResult(cp *ChargePoint, result *CallResult) {
	cp.pendingCallsMu.Lock()
	ch, ok := cp.pendingCalls[result.UniqueID]
	if ok {
		delete(cp.pendingCalls, result.UniqueID)
	}
	cp.pendingCallsMu.Unlock()

	if ok {
		ch <- result
	}
}

func (s *Server) handleBootNotification(cp *ChargePoint, call *Call) ([]byte, error) {
	var req BootNotificationRequest
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return CreateCallError(call.UniqueID, "FormationViolation", "Invalid request format", nil)
	}

	cp.mu.Lock()
	cp.Vendor = req.ChargePointVendor
	cp.Model = req.ChargePointModel
	cp.SerialNumber = req.ChargePointSerialNumber
	cp.FirmwareVersion = req.FirmwareVersion
	cp.mu.Unlock()

	log.Printf("Boot notification from %s: %s %s (SN: %s, FW: %s)",
		cp.ID, req.ChargePointVendor, req.ChargePointModel, req.ChargePointSerialNumber, req.FirmwareVersion)

	resp := BootNotificationResponse{
		Status:      RegistrationStatusAccepted,
		CurrentTime: time.Now().UTC().Format(time.RFC3339),
		Interval:    300,
	}

	return CreateCallResult(call.UniqueID, resp)
}

func (s *Server) handleHeartbeat(cp *ChargePoint, call *Call) ([]byte, error) {
	cp.mu.Lock()
	cp.LastHeartbeat = time.Now()
	cp.mu.Unlock()

	resp := HeartbeatResponse{
		CurrentTime: time.Now().UTC().Format(time.RFC3339),
	}

	return CreateCallResult(call.UniqueID, resp)
}

func (s *Server) handleStatusNotification(cp *ChargePoint, call *Call) ([]byte, error) {
	var req StatusNotificationRequest
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return CreateCallError(call.UniqueID, "FormationViolation", "Invalid request format", nil)
	}

	cp.mu.Lock()
	if req.ConnectorID == 0 {
		cp.Status = req.Status
	} else {
		cp.ConnectorStatus[req.ConnectorID] = req.Status
	}
	cp.mu.Unlock()

	log.Printf("Status from %s connector %d: %s (error: %s)", cp.ID, req.ConnectorID, req.Status, req.ErrorCode)

	return CreateCallResult(call.UniqueID, StatusNotificationResponse{})
}

func (s *Server) handleMeterValues(cp *ChargePoint, call *Call) ([]byte, error) {
	var req MeterValuesRequest
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return CreateCallError(call.UniqueID, "FormationViolation", "Invalid request format", nil)
	}

	cp.mu.Lock()
	cp.MeterValues[req.ConnectorID] = append(cp.MeterValues[req.ConnectorID], req.MeterValue...)
	if len(cp.MeterValues[req.ConnectorID]) > 100 {
		cp.MeterValues[req.ConnectorID] = cp.MeterValues[req.ConnectorID][len(cp.MeterValues[req.ConnectorID])-100:]
	}
	cp.mu.Unlock()

	for _, mv := range req.MeterValue {
		for _, sv := range mv.SampledValue {
			log.Printf("Meter value from %s connector %d: %s %s = %s",
				cp.ID, req.ConnectorID, sv.Measurand, sv.Unit, sv.Value)
		}
	}

	return CreateCallResult(call.UniqueID, MeterValuesResponse{})
}

func (s *Server) handleStartTransaction(cp *ChargePoint, call *Call) ([]byte, error) {
	var req StartTransactionRequest
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return CreateCallError(call.UniqueID, "FormationViolation", "Invalid request format", nil)
	}

	s.txMu.Lock()
	txID := s.nextTxID
	s.nextTxID++
	s.txMu.Unlock()

	tx := &Transaction{
		ID:          txID,
		ConnectorID: req.ConnectorID,
		IdTag:       req.IdTag,
		MeterStart:  req.MeterStart,
		StartTime:   time.Now(),
		Active:      true,
	}

	cp.mu.Lock()
	cp.Transactions[txID] = tx
	cp.mu.Unlock()

	log.Printf("Transaction started on %s connector %d: TX#%d, tag=%s, meter=%d",
		cp.ID, req.ConnectorID, txID, req.IdTag, req.MeterStart)

	resp := StartTransactionResponse{
		TransactionID: txID,
		IdTagInfo: IdTagInfo{
			Status: "Accepted",
		},
	}

	return CreateCallResult(call.UniqueID, resp)
}

func (s *Server) handleStopTransaction(cp *ChargePoint, call *Call) ([]byte, error) {
	var req StopTransactionRequest
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return CreateCallError(call.UniqueID, "FormationViolation", "Invalid request format", nil)
	}

	cp.mu.Lock()
	if tx, ok := cp.Transactions[req.TransactionID]; ok {
		tx.MeterStop = req.MeterStop
		tx.StopTime = time.Now()
		tx.Active = false
	}
	cp.mu.Unlock()

	log.Printf("Transaction stopped on %s: TX#%d, meter=%d, reason=%s",
		cp.ID, req.TransactionID, req.MeterStop, req.Reason)

	resp := StopTransactionResponse{
		IdTagInfo: &IdTagInfo{
			Status: "Accepted",
		},
	}

	return CreateCallResult(call.UniqueID, resp)
}

func (s *Server) handleAuthorize(cp *ChargePoint, call *Call) ([]byte, error) {
	var req AuthorizeRequest
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return CreateCallError(call.UniqueID, "FormationViolation", "Invalid request format", nil)
	}

	log.Printf("Authorization request from %s for tag: %s", cp.ID, req.IdTag)

	resp := AuthorizeResponse{
		IdTagInfo: IdTagInfo{
			Status: "Accepted",
		},
	}

	return CreateCallResult(call.UniqueID, resp)
}

func (s *Server) handleDataTransfer(cp *ChargePoint, call *Call) ([]byte, error) {
	var req DataTransferRequest
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return CreateCallError(call.UniqueID, "FormationViolation", "Invalid request format", nil)
	}

	log.Printf("Data transfer from %s: vendor=%s, messageId=%s, data=%s",
		cp.ID, req.VendorID, req.MessageID, req.Data)

	resp := DataTransferResponse{
		Status: "Accepted",
	}

	return CreateCallResult(call.UniqueID, resp)
}

// SendCall sends a Call message to a charge point and waits for the response
func (s *Server) SendCall(ctx context.Context, cpID, action string, payload any) (*CallResult, error) {
	cp, ok := s.GetChargePoint(cpID)
	if !ok {
		return nil, fmt.Errorf("charge point not found: %s", cpID)
	}

	uniqueID := uuid.New().String()
	msg, err := CreateCall(uniqueID, action, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create call: %w", err)
	}

	resultCh := make(chan *CallResult, 1)
	cp.pendingCallsMu.Lock()
	cp.pendingCalls[uniqueID] = resultCh
	cp.pendingCallsMu.Unlock()

	defer func() {
		cp.pendingCallsMu.Lock()
		delete(cp.pendingCalls, uniqueID)
		cp.pendingCallsMu.Unlock()
	}()

	log.Printf("Sending to %s: %s", cpID, string(msg))
	cp.mu.Lock()
	err = cp.conn.WriteMessage(websocket.TextMessage, msg)
	cp.mu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	select {
	case result := <-resultCh:
		return result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// UnlockConnector sends an UnlockConnector request
func (s *Server) UnlockConnector(ctx context.Context, cpID string, connectorID int) (string, error) {
	req := UnlockConnectorRequest{
		ConnectorID: connectorID,
	}

	result, err := s.SendCall(ctx, cpID, ActionUnlockConnector, req)
	if err != nil {
		return "", err
	}

	var resp UnlockConnectorResponse
	data, _ := json.Marshal(result.Payload)
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return resp.Status, nil
}

// ChangeAvailability sends a ChangeAvailability request
func (s *Server) ChangeAvailability(ctx context.Context, cpID string, connectorID int, availabilityType string) (string, error) {
	req := ChangeAvailabilityRequest{
		ConnectorID: connectorID,
		Type:        availabilityType,
	}

	result, err := s.SendCall(ctx, cpID, ActionChangeAvailability, req)
	if err != nil {
		return "", err
	}

	var resp ChangeAvailabilityResponse
	data, _ := json.Marshal(result.Payload)
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return resp.Status, nil
}

// RemoteStartTransaction sends a RemoteStartTransaction request
func (s *Server) RemoteStartTransaction(ctx context.Context, cpID string, connectorID int, idTag string) (string, error) {
	req := RemoteStartTransactionRequest{
		ConnectorID: connectorID,
		IdTag:       idTag,
	}

	result, err := s.SendCall(ctx, cpID, ActionRemoteStartTransaction, req)
	if err != nil {
		return "", err
	}

	var resp RemoteStartTransactionResponse
	data, _ := json.Marshal(result.Payload)
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return resp.Status, nil
}

// RemoteStopTransaction sends a RemoteStopTransaction request
func (s *Server) RemoteStopTransaction(ctx context.Context, cpID string, transactionID int) (string, error) {
	req := RemoteStopTransactionRequest{
		TransactionID: transactionID,
	}

	result, err := s.SendCall(ctx, cpID, ActionRemoteStopTransaction, req)
	if err != nil {
		return "", err
	}

	var resp RemoteStopTransactionResponse
	data, _ := json.Marshal(result.Payload)
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return resp.Status, nil
}

// GetConfiguration sends a GetConfiguration request
func (s *Server) GetConfiguration(ctx context.Context, cpID string, keys []string) (*GetConfigurationResponse, error) {
	req := GetConfigurationRequest{
		Key: keys,
	}

	result, err := s.SendCall(ctx, cpID, ActionGetConfiguration, req)
	if err != nil {
		return nil, err
	}

	var resp GetConfigurationResponse
	data, _ := json.Marshal(result.Payload)
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &resp, nil
}
