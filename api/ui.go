package api

import "net/http"

// RegisterUI registers the web UI handler
func RegisterUI(mux *http.ServeMux) {
	mux.HandleFunc("/ui", serveUI)
	mux.HandleFunc("/ui/", serveUI)
}

func serveUI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(uiHTML))
}

const uiHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>OCPP Central System</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: #1a1a2e;
            color: #eee;
            min-height: 100vh;
            padding: 20px;
        }
        h1 {
            color: #00d9ff;
            margin-bottom: 20px;
            font-size: 1.8rem;
        }
        h2 {
            color: #888;
            font-size: 0.9rem;
            text-transform: uppercase;
            letter-spacing: 1px;
            margin-bottom: 15px;
        }
        .container { max-width: 1200px; margin: 0 auto; }
        .no-chargers {
            background: #252542;
            border-radius: 12px;
            padding: 40px;
            text-align: center;
            color: #666;
        }
        .charger-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
            gap: 20px;
        }
        .charger-card {
            background: #252542;
            border-radius: 12px;
            padding: 20px;
            border: 1px solid #333;
        }
        .charger-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 15px;
        }
        .charger-id {
            font-size: 1.3rem;
            font-weight: 600;
            color: #00d9ff;
        }
        .status-badge {
            padding: 4px 12px;
            border-radius: 20px;
            font-size: 0.8rem;
            font-weight: 500;
        }
        .status-available { background: #00c853; color: #000; }
        .status-charging { background: #ffab00; color: #000; }
        .status-preparing { background: #2196f3; color: #fff; }
        .status-finishing { background: #9c27b0; color: #fff; }
        .status-unavailable { background: #666; color: #fff; }
        .status-faulted { background: #f44336; color: #fff; }
        .status-unknown { background: #444; color: #fff; }
        .charger-info {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 10px;
            margin-bottom: 15px;
            font-size: 0.9rem;
        }
        .info-label { color: #666; }
        .info-value { color: #ccc; }
        .connector-section {
            background: #1a1a2e;
            border-radius: 8px;
            padding: 12px;
            margin-bottom: 15px;
        }
        .connector-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 10px;
        }
        .connector-title { font-weight: 500; }
        .meter-values {
            display: grid;
            grid-template-columns: repeat(2, 1fr);
            gap: 8px;
            font-size: 0.85rem;
        }
        .meter-item {
            background: #252542;
            padding: 8px;
            border-radius: 6px;
        }
        .meter-label { color: #666; font-size: 0.75rem; }
        .meter-value { color: #00d9ff; font-weight: 600; }
        .actions {
            display: flex;
            gap: 10px;
            flex-wrap: wrap;
        }
        button {
            padding: 10px 20px;
            border: none;
            border-radius: 8px;
            cursor: pointer;
            font-size: 0.9rem;
            font-weight: 500;
            transition: all 0.2s;
        }
        button:hover { transform: translateY(-1px); }
        button:active { transform: translateY(0); }
        .btn-unlock { background: #00c853; color: #000; }
        .btn-lock { background: #f44336; color: #fff; }
        .btn-start { background: #2196f3; color: #fff; }
        .btn-stop { background: #ff5722; color: #fff; }
        .btn-refresh { background: #333; color: #fff; }
        button:disabled {
            opacity: 0.5;
            cursor: not-allowed;
            transform: none;
        }
        .toast {
            position: fixed;
            bottom: 20px;
            right: 20px;
            padding: 15px 25px;
            border-radius: 8px;
            color: #fff;
            font-weight: 500;
            animation: slideIn 0.3s ease;
            z-index: 1000;
        }
        .toast-success { background: #00c853; }
        .toast-error { background: #f44336; }
        @keyframes slideIn {
            from { transform: translateX(100px); opacity: 0; }
            to { transform: translateX(0); opacity: 1; }
        }
        .transactions {
            margin-top: 10px;
            font-size: 0.85rem;
        }
        .transaction {
            background: #1a1a2e;
            padding: 8px 12px;
            border-radius: 6px;
            margin-top: 8px;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        .tx-active { border-left: 3px solid #00c853; }
        .tx-stopped { border-left: 3px solid #666; }
        .auto-refresh {
            display: flex;
            align-items: center;
            gap: 10px;
            margin-bottom: 20px;
        }
        .auto-refresh label {
            display: flex;
            align-items: center;
            gap: 5px;
            cursor: pointer;
        }
        .last-update { color: #666; font-size: 0.8rem; }
    </style>
</head>
<body>
    <div class="container">
        <h1>OCPP Central System</h1>
        <div class="auto-refresh">
            <label>
                <input type="checkbox" id="autoRefresh" checked>
                Auto-refresh (5s)
            </label>
            <button class="btn-refresh" onclick="loadChargePoints()">Refresh Now</button>
            <span class="last-update" id="lastUpdate"></span>
        </div>
        <h2>Connected Charge Points</h2>
        <div id="chargePoints" class="no-chargers">
            Loading...
        </div>
    </div>

    <script>
        let refreshInterval;

        async function loadChargePoints() {
            try {
                const res = await fetch('/api/chargepoints');
                const chargePoints = await res.json();
                renderChargePoints(chargePoints);
                document.getElementById('lastUpdate').textContent = 'Updated: ' + new Date().toLocaleTimeString();
            } catch (err) {
                showToast('Failed to load charge points', 'error');
            }
        }

        function renderChargePoints(chargePoints) {
            const container = document.getElementById('chargePoints');

            if (!chargePoints || chargePoints.length === 0) {
                container.innerHTML = '<div class="no-chargers">No charge points connected.<br><br>Configure your Wallbox OCPP URL to:<br><code>ws://' + location.hostname + ':8080/ocpp/YOUR_CHARGER_ID</code></div>';
                return;
            }

            container.className = 'charger-grid';
            container.innerHTML = chargePoints.map(cp => renderChargePoint(cp)).join('');
        }

        function renderChargePoint(cp) {
            const status = cp.status || 'Unknown';
            const statusClass = 'status-' + status.toLowerCase();

            const connectors = Object.entries(cp.connectorStatus || {}).map(([id, st]) =>
                '<div class="connector-section">' +
                    '<div class="connector-header">' +
                        '<span class="connector-title">Connector ' + id + '</span>' +
                        '<span class="status-badge status-' + st.toLowerCase() + '">' + st + '</span>' +
                    '</div>' +
                    '<div class="actions">' +
                        '<button class="btn-unlock" onclick="unlock(\'' + cp.id + '\', ' + id + ')">Unlock</button>' +
                        '<button class="btn-lock" onclick="lock(\'' + cp.id + '\', ' + id + ')">Lock</button>' +
                        '<button class="btn-start" onclick="startCharge(\'' + cp.id + '\', ' + id + ')">Start</button>' +
                    '</div>' +
                '</div>'
            ).join('');

            const defaultConnector = Object.keys(cp.connectorStatus || {}).length === 0 ?
                '<div class="connector-section">' +
                    '<div class="connector-header">' +
                        '<span class="connector-title">Connector 1</span>' +
                        '<span class="status-badge status-unknown">Waiting</span>' +
                    '</div>' +
                    '<div class="actions">' +
                        '<button class="btn-unlock" onclick="unlock(\'' + cp.id + '\', 1)">Unlock</button>' +
                        '<button class="btn-lock" onclick="lock(\'' + cp.id + '\', 1)">Lock</button>' +
                        '<button class="btn-start" onclick="startCharge(\'' + cp.id + '\', 1)">Start</button>' +
                    '</div>' +
                '</div>' : '';

            const transactions = (cp.transactions || []).map(tx =>
                '<div class="transaction ' + (tx.active ? 'tx-active' : 'tx-stopped') + '">' +
                    '<div>' +
                        '<div>TX #' + tx.id + ' - ' + (tx.active ? 'Active' : 'Stopped') + '</div>' +
                        '<div style="color:#666;font-size:0.75rem">' + (tx.energyWh ? (tx.energyWh / 1000).toFixed(2) + ' kWh' : 'Started ' + new Date(tx.startTime).toLocaleTimeString()) + '</div>' +
                    '</div>' +
                    (tx.active ? '<button class="btn-stop" onclick="stopCharge(\'' + cp.id + '\', ' + tx.id + ')">Stop</button>' : '') +
                '</div>'
            ).join('');

            return '<div class="charger-card">' +
                '<div class="charger-header">' +
                    '<span class="charger-id">' + cp.id + '</span>' +
                    '<span class="status-badge ' + statusClass + '">' + status + '</span>' +
                '</div>' +
                '<div class="charger-info">' +
                    '<div><span class="info-label">Vendor</span></div>' +
                    '<div><span class="info-value">' + (cp.vendor || '-') + '</span></div>' +
                    '<div><span class="info-label">Model</span></div>' +
                    '<div><span class="info-value">' + (cp.model || '-') + '</span></div>' +
                    '<div><span class="info-label">Serial</span></div>' +
                    '<div><span class="info-value">' + (cp.serialNumber || '-') + '</span></div>' +
                    '<div><span class="info-label">Firmware</span></div>' +
                    '<div><span class="info-value">' + (cp.firmwareVersion || '-') + '</span></div>' +
                '</div>' +
                (connectors || defaultConnector) +
                (transactions ? '<div class="transactions"><strong>Transactions</strong>' + transactions + '</div>' : '') +
            '</div>';
        }

        async function unlock(cpId, connector) {
            await sendCommand(cpId, 'unlock', { connector });
        }

        async function lock(cpId, connector) {
            await sendCommand(cpId, 'lock', { connector });
        }

        async function startCharge(cpId, connector) {
            const tag = prompt('Enter ID Tag (or leave empty for default):', 'REMOTE');
            if (tag === null) return;
            await sendCommand(cpId, 'start', { connector, tag: tag || 'REMOTE' });
        }

        async function stopCharge(cpId, txId) {
            await sendCommand(cpId, 'stop', { transaction: txId });
        }

        async function sendCommand(cpId, action, params) {
            try {
                const query = new URLSearchParams(params).toString();
                const res = await fetch('/api/chargepoints/' + cpId + '/' + action + '?' + query, {
                    method: 'POST'
                });
                const result = await res.json();

                if (result.error) {
                    showToast(result.error, 'error');
                } else {
                    showToast(action + ': ' + (result.status || 'OK'), 'success');
                    setTimeout(loadChargePoints, 500);
                }
            } catch (err) {
                showToast('Command failed: ' + err.message, 'error');
            }
        }

        function showToast(message, type) {
            const toast = document.createElement('div');
            toast.className = 'toast toast-' + type;
            toast.textContent = message;
            document.body.appendChild(toast);
            setTimeout(() => toast.remove(), 3000);
        }

        function setupAutoRefresh() {
            const checkbox = document.getElementById('autoRefresh');

            function updateRefresh() {
                if (refreshInterval) clearInterval(refreshInterval);
                if (checkbox.checked) {
                    refreshInterval = setInterval(loadChargePoints, 5000);
                }
            }

            checkbox.addEventListener('change', updateRefresh);
            updateRefresh();
        }

        loadChargePoints();
        setupAutoRefresh();
    </script>
</body>
</html>`
