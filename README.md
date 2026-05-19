# Go OCPP Server

OCPP 1.6-J (JSON over WebSocket) Central System for controlling EV chargers like the Wallbox 3.

## Build

```bash
go build -o ocpp-server .
```

## Run

```bash
./ocpp-server
```

Options:
- `-ocpp-port`: WebSocket port for charger connections (default: 8080)
- `-api-port`: HTTP API port (default: 8081)

## Configure Your Wallbox

In your Wallbox settings, configure OCPP with:
- URL: `ws://YOUR_SERVER_IP:8080/ocpp/CHARGER_ID`
- Replace `CHARGER_ID` with a unique identifier for your charger (e.g., `wallbox1`)

## API Endpoints

### List Charge Points
```bash
curl http://localhost:8081/api/chargepoints
```

### Get Charge Point Details
```bash
curl http://localhost:8081/api/chargepoints/wallbox1
```

### Get Status
```bash
curl http://localhost:8081/api/chargepoints/wallbox1/status
```

### Get Meter Values
```bash
curl http://localhost:8081/api/chargepoints/wallbox1/meter
```

### Unlock Connector
```bash
curl -X POST "http://localhost:8081/api/chargepoints/wallbox1/unlock?connector=1"
```

### Lock Connector
```bash
curl -X POST "http://localhost:8081/api/chargepoints/wallbox1/lock?connector=1"
```

### Start Charging Remotely
```bash
curl -X POST "http://localhost:8081/api/chargepoints/wallbox1/start?connector=1&tag=MYTAG"
```

### Stop Charging
```bash
curl -X POST "http://localhost:8081/api/chargepoints/wallbox1/stop?transaction=1"
```

### Get Configuration
```bash
curl http://localhost:8081/api/chargepoints/wallbox1/config
```

## Supported OCPP 1.6 Messages

### Incoming (Charger → Server)
- BootNotification
- Heartbeat
- StatusNotification
- MeterValues
- StartTransaction
- StopTransaction
- Authorize
- DataTransfer

### Outgoing (Server → Charger)
- UnlockConnector
- ChangeAvailability
- RemoteStartTransaction
- RemoteStopTransaction
- GetConfiguration
