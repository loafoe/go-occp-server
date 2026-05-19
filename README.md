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
- `-credentials-file`: Path to JSON file with charge point credentials

## Authentication

The server supports Basic Auth for charge point authentication. Credentials can be provided via:

### Option 1: Credentials file
```bash
./ocpp-server --credentials-file /path/to/credentials.json
```

Format of `credentials.json`:
```json
{
  "wallbox1": "secretpassword123",
  "wallbox2": "anotherpassword456"
}
```

### Option 2: Environment variable
```bash
export OCPP_CREDENTIALS='{"wallbox1":"secretpassword123"}'
./ocpp-server
```

### Wallbox Configuration
In your Wallbox OCPP settings:
- URL: `ws://YOUR_SERVER_IP:8080/ocpp/wallbox1`
- Username: `wallbox1` (must match the charge point ID in the URL)
- Password: `secretpassword123`

If no credentials are configured, the server runs in open mode (any charger can connect).

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

## Kubernetes Deployment

A Helm chart is available at https://loafoe.github.io/helm-charts

```bash
helm repo add loafoe https://loafoe.github.io/helm-charts
helm install ocpp loafoe/go-ocpp-server \
  --set auth.enabled=true \
  --set auth.credentials.wallbox1=secretpassword123 \
  --set httpRoute.enabled=true \
  --set httpRoute.hostname=ocpp.example.com
```

Or use auto-provisioning to generate credentials:
```bash
helm install ocpp loafoe/go-ocpp-server \
  --set auth.enabled=true \
  --set provisioning.enabled=true \
  --set provisioning.count=1 \
  --set provisioning.idPrefix=wallbox
```

Then retrieve the generated credentials:
```bash
kubectl get secret ocpp-go-ocpp-server-credentials -o jsonpath='{.data.credentials\.json}' | base64 -d
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
