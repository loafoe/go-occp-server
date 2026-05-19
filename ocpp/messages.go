package ocpp

import (
	"encoding/json"
	"fmt"
)

// OCPP 1.6 message types
const (
	MessageTypeCall       = 2
	MessageTypeCallResult = 3
	MessageTypeCallError  = 4
)

// OCPP 1.6 Actions
const (
	ActionBootNotification       = "BootNotification"
	ActionHeartbeat              = "Heartbeat"
	ActionStatusNotification     = "StatusNotification"
	ActionMeterValues            = "MeterValues"
	ActionStartTransaction       = "StartTransaction"
	ActionStopTransaction        = "StopTransaction"
	ActionAuthorize              = "Authorize"
	ActionDataTransfer           = "DataTransfer"
	ActionUnlockConnector        = "UnlockConnector"
	ActionChangeAvailability     = "ChangeAvailability"
	ActionGetConfiguration       = "GetConfiguration"
	ActionRemoteStartTransaction = "RemoteStartTransaction"
	ActionRemoteStopTransaction  = "RemoteStopTransaction"
)

// Registration status
const (
	RegistrationStatusAccepted = "Accepted"
	RegistrationStatusPending  = "Pending"
	RegistrationStatusRejected = "Rejected"
)

// Availability types
const (
	AvailabilityTypeInoperative = "Inoperative"
	AvailabilityTypeOperative   = "Operative"
)

// Availability status
const (
	AvailabilityStatusAccepted  = "Accepted"
	AvailabilityStatusRejected  = "Rejected"
	AvailabilityStatusScheduled = "Scheduled"
)

// Unlock status
const (
	UnlockStatusUnlocked     = "Unlocked"
	UnlockStatusUnlockFailed = "UnlockFailed"
	UnlockStatusNotSupported = "NotSupported"
)

// Charge point status
const (
	ChargePointStatusAvailable   = "Available"
	ChargePointStatusPreparing   = "Preparing"
	ChargePointStatusCharging    = "Charging"
	ChargePointStatusSuspendedEV = "SuspendedEV"
	ChargePointStatusFinishing   = "Finishing"
	ChargePointStatusReserved    = "Reserved"
	ChargePointStatusUnavailable = "Unavailable"
	ChargePointStatusFaulted     = "Faulted"
)

// Remote start/stop status
const (
	RemoteStartStopStatusAccepted = "Accepted"
	RemoteStartStopStatusRejected = "Rejected"
)

// Call represents an OCPP Call message [2, uniqueId, action, payload]
type Call struct {
	MessageTypeID int
	UniqueID      string
	Action        string
	Payload       json.RawMessage
}

// CallResult represents an OCPP CallResult message [3, uniqueId, payload]
type CallResult struct {
	MessageTypeID int
	UniqueID      string
	Payload       any
}

// CallError represents an OCPP CallError message [4, uniqueId, errorCode, errorDescription, errorDetails]
type CallError struct {
	MessageTypeID    int
	UniqueID         string
	ErrorCode        string
	ErrorDescription string
	ErrorDetails     any
}

// ParseMessage parses a raw OCPP message
func ParseMessage(data []byte) (any, error) {
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("invalid OCPP message format: %w", err)
	}

	if len(raw) < 3 {
		return nil, fmt.Errorf("invalid OCPP message: expected at least 3 elements")
	}

	var messageType int
	if err := json.Unmarshal(raw[0], &messageType); err != nil {
		return nil, fmt.Errorf("invalid message type: %w", err)
	}

	var uniqueID string
	if err := json.Unmarshal(raw[1], &uniqueID); err != nil {
		return nil, fmt.Errorf("invalid unique ID: %w", err)
	}

	switch messageType {
	case MessageTypeCall:
		if len(raw) < 4 {
			return nil, fmt.Errorf("invalid Call message: expected 4 elements")
		}
		var action string
		if err := json.Unmarshal(raw[2], &action); err != nil {
			return nil, fmt.Errorf("invalid action: %w", err)
		}
		return &Call{
			MessageTypeID: messageType,
			UniqueID:      uniqueID,
			Action:        action,
			Payload:       raw[3],
		}, nil

	case MessageTypeCallResult:
		return &CallResult{
			MessageTypeID: messageType,
			UniqueID:      uniqueID,
			Payload:       raw[2],
		}, nil

	case MessageTypeCallError:
		if len(raw) < 5 {
			return nil, fmt.Errorf("invalid CallError message: expected 5 elements")
		}
		var errorCode, errorDesc string
		_ = json.Unmarshal(raw[2], &errorCode)
		_ = json.Unmarshal(raw[3], &errorDesc)
		return &CallError{
			MessageTypeID:    messageType,
			UniqueID:         uniqueID,
			ErrorCode:        errorCode,
			ErrorDescription: errorDesc,
			ErrorDetails:     raw[4],
		}, nil

	default:
		return nil, fmt.Errorf("unknown message type: %d", messageType)
	}
}

// CreateCall creates an OCPP Call message
func CreateCall(uniqueID, action string, payload any) ([]byte, error) {
	msg := []any{MessageTypeCall, uniqueID, action, payload}
	return json.Marshal(msg)
}

// CreateCallResult creates an OCPP CallResult message
func CreateCallResult(uniqueID string, payload any) ([]byte, error) {
	msg := []any{MessageTypeCallResult, uniqueID, payload}
	return json.Marshal(msg)
}

// CreateCallError creates an OCPP CallError message
func CreateCallError(uniqueID, errorCode, errorDesc string, details any) ([]byte, error) {
	msg := []any{MessageTypeCallError, uniqueID, errorCode, errorDesc, details}
	return json.Marshal(msg)
}

// BootNotificationRequest represents a BootNotification.req
type BootNotificationRequest struct {
	ChargePointVendor       string `json:"chargePointVendor"`
	ChargePointModel        string `json:"chargePointModel"`
	ChargePointSerialNumber string `json:"chargePointSerialNumber,omitempty"`
	ChargeBoxSerialNumber   string `json:"chargeBoxSerialNumber,omitempty"`
	FirmwareVersion         string `json:"firmwareVersion,omitempty"`
	Iccid                   string `json:"iccid,omitempty"`
	Imsi                    string `json:"imsi,omitempty"`
	MeterType               string `json:"meterType,omitempty"`
	MeterSerialNumber       string `json:"meterSerialNumber,omitempty"`
}

// BootNotificationResponse represents a BootNotification.conf
type BootNotificationResponse struct {
	Status      string `json:"status"`
	CurrentTime string `json:"currentTime"`
	Interval    int    `json:"interval"`
}

// HeartbeatRequest represents a Heartbeat.req
type HeartbeatRequest struct{}

// HeartbeatResponse represents a Heartbeat.conf
type HeartbeatResponse struct {
	CurrentTime string `json:"currentTime"`
}

// StatusNotificationRequest represents a StatusNotification.req
type StatusNotificationRequest struct {
	ConnectorID     int    `json:"connectorId"`
	ErrorCode       string `json:"errorCode"`
	Status          string `json:"status"`
	Timestamp       string `json:"timestamp,omitempty"`
	Info            string `json:"info,omitempty"`
	VendorID        string `json:"vendorId,omitempty"`
	VendorErrorCode string `json:"vendorErrorCode,omitempty"`
}

// StatusNotificationResponse represents a StatusNotification.conf
type StatusNotificationResponse struct{}

// MeterValuesRequest represents a MeterValues.req
type MeterValuesRequest struct {
	ConnectorID   int          `json:"connectorId"`
	TransactionID int          `json:"transactionId,omitempty"`
	MeterValue    []MeterValue `json:"meterValue"`
}

// MeterValue represents a meter value sample
type MeterValue struct {
	Timestamp    string         `json:"timestamp"`
	SampledValue []SampledValue `json:"sampledValue"`
}

// SampledValue represents a sampled value
type SampledValue struct {
	Value     string `json:"value"`
	Context   string `json:"context,omitempty"`
	Format    string `json:"format,omitempty"`
	Measurand string `json:"measurand,omitempty"`
	Phase     string `json:"phase,omitempty"`
	Location  string `json:"location,omitempty"`
	Unit      string `json:"unit,omitempty"`
}

// MeterValuesResponse represents a MeterValues.conf
type MeterValuesResponse struct{}

// StartTransactionRequest represents a StartTransaction.req
type StartTransactionRequest struct {
	ConnectorID   int    `json:"connectorId"`
	IdTag         string `json:"idTag"`
	MeterStart    int    `json:"meterStart"`
	Timestamp     string `json:"timestamp"`
	ReservationID int    `json:"reservationId,omitempty"`
}

// StartTransactionResponse represents a StartTransaction.conf
type StartTransactionResponse struct {
	TransactionID int     `json:"transactionId"`
	IdTagInfo     IdTagInfo `json:"idTagInfo"`
}

// StopTransactionRequest represents a StopTransaction.req
type StopTransactionRequest struct {
	MeterStop         int          `json:"meterStop"`
	Timestamp         string       `json:"timestamp"`
	TransactionID     int          `json:"transactionId"`
	IdTag             string       `json:"idTag,omitempty"`
	Reason            string       `json:"reason,omitempty"`
	TransactionData   []MeterValue `json:"transactionData,omitempty"`
}

// StopTransactionResponse represents a StopTransaction.conf
type StopTransactionResponse struct {
	IdTagInfo *IdTagInfo `json:"idTagInfo,omitempty"`
}

// IdTagInfo represents authorization info for an ID tag
type IdTagInfo struct {
	Status      string `json:"status"`
	ExpiryDate  string `json:"expiryDate,omitempty"`
	ParentIdTag string `json:"parentIdTag,omitempty"`
}

// AuthorizeRequest represents an Authorize.req
type AuthorizeRequest struct {
	IdTag string `json:"idTag"`
}

// AuthorizeResponse represents an Authorize.conf
type AuthorizeResponse struct {
	IdTagInfo IdTagInfo `json:"idTagInfo"`
}

// UnlockConnectorRequest represents an UnlockConnector.req
type UnlockConnectorRequest struct {
	ConnectorID int `json:"connectorId"`
}

// UnlockConnectorResponse represents an UnlockConnector.conf
type UnlockConnectorResponse struct {
	Status string `json:"status"`
}

// ChangeAvailabilityRequest represents a ChangeAvailability.req
type ChangeAvailabilityRequest struct {
	ConnectorID int    `json:"connectorId"`
	Type        string `json:"type"`
}

// ChangeAvailabilityResponse represents a ChangeAvailability.conf
type ChangeAvailabilityResponse struct {
	Status string `json:"status"`
}

// GetConfigurationRequest represents a GetConfiguration.req
type GetConfigurationRequest struct {
	Key []string `json:"key,omitempty"`
}

// GetConfigurationResponse represents a GetConfiguration.conf
type GetConfigurationResponse struct {
	ConfigurationKey []ConfigurationKey `json:"configurationKey,omitempty"`
	UnknownKey       []string           `json:"unknownKey,omitempty"`
}

// ConfigurationKey represents a configuration key-value pair
type ConfigurationKey struct {
	Key      string `json:"key"`
	Readonly bool   `json:"readonly"`
	Value    string `json:"value,omitempty"`
}

// RemoteStartTransactionRequest represents a RemoteStartTransaction.req
type RemoteStartTransactionRequest struct {
	ConnectorID     int             `json:"connectorId,omitempty"`
	IdTag           string          `json:"idTag"`
	ChargingProfile *ChargingProfile `json:"chargingProfile,omitempty"`
}

// RemoteStartTransactionResponse represents a RemoteStartTransaction.conf
type RemoteStartTransactionResponse struct {
	Status string `json:"status"`
}

// RemoteStopTransactionRequest represents a RemoteStopTransaction.req
type RemoteStopTransactionRequest struct {
	TransactionID int `json:"transactionId"`
}

// RemoteStopTransactionResponse represents a RemoteStopTransaction.conf
type RemoteStopTransactionResponse struct {
	Status string `json:"status"`
}

// ChargingProfile represents a charging profile
type ChargingProfile struct {
	ChargingProfileID      int                     `json:"chargingProfileId"`
	TransactionID          int                     `json:"transactionId,omitempty"`
	StackLevel             int                     `json:"stackLevel"`
	ChargingProfilePurpose string                  `json:"chargingProfilePurpose"`
	ChargingProfileKind    string                  `json:"chargingProfileKind"`
	RecurrencyKind         string                  `json:"recurrencyKind,omitempty"`
	ValidFrom              string                  `json:"validFrom,omitempty"`
	ValidTo                string                  `json:"validTo,omitempty"`
	ChargingSchedule       ChargingSchedule        `json:"chargingSchedule"`
}

// ChargingSchedule represents a charging schedule
type ChargingSchedule struct {
	Duration               int                      `json:"duration,omitempty"`
	StartSchedule          string                   `json:"startSchedule,omitempty"`
	ChargingRateUnit       string                   `json:"chargingRateUnit"`
	ChargingSchedulePeriod []ChargingSchedulePeriod `json:"chargingSchedulePeriod"`
	MinChargingRate        float64                  `json:"minChargingRate,omitempty"`
}

// ChargingSchedulePeriod represents a period in a charging schedule
type ChargingSchedulePeriod struct {
	StartPeriod  int     `json:"startPeriod"`
	Limit        float64 `json:"limit"`
	NumberPhases int     `json:"numberPhases,omitempty"`
}

// DataTransferRequest represents a DataTransfer.req
type DataTransferRequest struct {
	VendorID  string `json:"vendorId"`
	MessageID string `json:"messageId,omitempty"`
	Data      string `json:"data,omitempty"`
}

// DataTransferResponse represents a DataTransfer.conf
type DataTransferResponse struct {
	Status string `json:"status"`
	Data   string `json:"data,omitempty"`
}
