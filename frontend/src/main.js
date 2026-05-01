import {Events} from "@wailsio/runtime";
import {NetService} from "../bindings/github.com/huaky-tec-fornb/go-net-tool/internal/service";

// --- DOM References ---
const $ = (id) => document.getElementById(id);

const els = {
    protocol: $("protocol"),
    localIp: $("local-ip"),
    localPort: $("local-port"),
    remoteIp: $("remote-ip"),
    remotePort: $("remote-port"),
    remoteRow: $("remote-row"),
    btnConnect: $("btn-connect"),
    btnDisconnect: $("btn-disconnect"),
    statusBadge: $("status-badge"),
    clientBar: $("client-bar"),
    clientSelect: $("client-select"),
    btnDisconnectClient: $("btn-disconnect-client"),
    receivedDisplay: $("received-display"),
    displayMode: $("display-mode"),
    btnClear: $("btn-clear"),
    autoScroll: $("auto-scroll"),
    sentCounter: $("sent-counter"),
    receivedCounter: $("received-counter"),
    sendInput: $("send-input"),
    btnSend: $("btn-send"),
    btnSendHex: $("btn-send-hex"),
    autoSend: $("auto-send"),
    autoSendInterval: $("auto-send-interval"),
};

let autoSendTimer = null;
let selectedClientId = "";

// --- Protocol Change ---
els.protocol.addEventListener("change", () => {
    const proto = els.protocol.value;
    if (proto === "tcp-server") {
        els.remoteRow.classList.add("hidden");
        els.clientBar.style.display = "block";
    } else {
        els.remoteRow.classList.remove("hidden");
        els.clientBar.style.display = "none";
    }
});

// --- Connect / Disconnect ---
els.btnConnect.addEventListener("click", async () => {
    const config = {
        protocol: els.protocol.value,
        localIp: els.localIp.value,
        localPort: parseInt(els.localPort.value) || 0,
        remoteIp: els.remoteIp.value,
        remotePort: parseInt(els.remotePort.value) || 0,
    };

    els.btnConnect.disabled = true;
    setStatus("connecting", "连接中...");

    try {
        const result = await NetService.Connect(config);
        if (result.startsWith("error:")) {
            setStatus("error", result.replace("error:", ""));
            els.btnConnect.disabled = false;
            appendSystem(result.replace("error: ", ""));
        } else {
            els.btnConnect.disabled = true;
            els.btnDisconnect.disabled = false;
            els.btnSend.disabled = false;
            els.btnSendHex.disabled = false;
            els.protocol.disabled = true;
            els.localIp.disabled = true;
            els.localPort.disabled = true;
            // UDP keeps remote fields editable for dynamic target changes
            if (config.protocol !== "udp") {
                els.remoteIp.disabled = true;
                els.remotePort.disabled = true;
            }
        }
    } catch (err) {
        setStatus("error", "连接异常");
        els.btnConnect.disabled = false;
        appendSystem("连接错误: " + err);
    }
});

els.btnDisconnect.addEventListener("click", async () => {
    try {
        await NetService.Disconnect();
    } catch (err) {
        console.error(err);
    }
    resetUI();
});

// --- Send ---
els.btnSend.addEventListener("click", async () => {
    const text = els.sendInput.value;
    if (!text) return;

    let result;
    if (selectedClientId && els.protocol.value === "tcp-server") {
        result = await NetService.SendToClient(selectedClientId, text);
    } else {
        // For UDP, update remote addr before each send
        if (els.protocol.value === "udp") {
            await NetService.SetRemoteAddr(els.remoteIp.value, parseInt(els.remotePort.value) || 0);
        }
        result = await NetService.Send(text);
    }
    if (result.startsWith("error:")) {
        appendSystem("发送失败: " + result.replace("error: ", ""));
    }
});

els.btnSendHex.addEventListener("click", async () => {
    const hexInput = els.sendInput.value.trim();
    if (!hexInput) return;

    // Validate hex
    const cleaned = hexInput.replace(/\s+/g, "");
    if (!/^[0-9a-fA-F]+$/.test(cleaned)) {
        appendSystem("无效的十六进制输入");
        return;
    }

    let result;
    if (selectedClientId && els.protocol.value === "tcp-server") {
        result = await NetService.SendHexToClient(selectedClientId, hexInput);
    } else {
        // For UDP, update remote addr before each send
        if (els.protocol.value === "udp") {
            await NetService.SetRemoteAddr(els.remoteIp.value, parseInt(els.remotePort.value) || 0);
        }
        result = await NetService.SendHex(hexInput);
    }
    if (result.startsWith("error:")) {
        appendSystem("发送失败: " + result.replace("error: ", ""));
    }
});

// --- Auto Send ---
els.autoSend.addEventListener("change", () => {
    if (els.autoSend.checked) {
        const interval = parseInt(els.autoSendInterval.value) || 1000;
        autoSendTimer = setInterval(() => {
            els.btnSend.click();
        }, interval);
    } else {
        clearInterval(autoSendTimer);
        autoSendTimer = null;
    }
});

els.autoSendInterval.addEventListener("change", () => {
    if (els.autoSend.checked) {
        clearInterval(autoSendTimer);
        const interval = parseInt(els.autoSendInterval.value) || 1000;
        autoSendTimer = setInterval(() => {
            els.btnSend.click();
        }, interval);
    }
});

// --- Display Mode ---
els.displayMode.addEventListener("change", () => {
    NetService.SetDisplayMode(els.displayMode.value);
});

// --- Clear ---
els.btnClear.addEventListener("click", () => {
    els.receivedDisplay.value = "";
    NetService.ClearCounters();
    els.sentCounter.textContent = "0";
    els.receivedCounter.textContent = "0";
});

// --- Client Selection (TCP Server) ---
els.clientSelect.addEventListener("change", () => {
    selectedClientId = els.clientSelect.value;
    els.btnDisconnectClient.disabled = !selectedClientId;
});

els.btnDisconnectClient.addEventListener("click", async () => {
    if (!selectedClientId) return;
    await NetService.DisconnectClient(selectedClientId);
    selectedClientId = "";
    els.btnDisconnectClient.disabled = true;
});

// --- Keyboard shortcut: Ctrl+Enter to send ---
els.sendInput.addEventListener("keydown", (e) => {
    if ((e.ctrlKey || e.metaKey) && e.key === "Enter") {
        e.preventDefault();
        els.btnSend.click();
    }
});

// --- Event Listeners from Go ---
Events.On("rx-data", (evt) => {
    const data = evt.data;
    const prefix = data.error
        ? `[${data.timestamp}] [系统] `
        : `[${data.timestamp}] [${data.direction}] `;

    let addr = "";
    if (data.srcAddr) addr += ` 来源:${data.srcAddr}`;
    if (data.dstAddr) addr += ` 目标:${data.dstAddr}`;

    const sizeInfo = data.size > 0 ? ` [${data.size}字节]` : "";
    const line = prefix + data.text + addr + sizeInfo + "\n";

    // Append to display
    const display = els.receivedDisplay;
    if (display.value.length > 50000) {
        display.value = display.value.slice(-25000);
    }
    display.value += line;

    // Auto scroll
    if (els.autoScroll.checked) {
        display.scrollTop = display.scrollHeight;
    }
});

Events.On("state-change", (evt) => {
    updateStatusUI(evt.data.state);
});

Events.On("stats-update", (evt) => {
    const stats = evt.data;
    els.sentCounter.textContent = (stats.bytesSent || 0).toLocaleString();
    els.receivedCounter.textContent = (stats.bytesReceived || 0).toLocaleString();
});

Events.On("clients-update", (evt) => {
    const clients = evt.data || [];
    const select = els.clientSelect;
    select.innerHTML = '<option value="">-- 选择客户端 --</option>';
    clients.forEach((c) => {
        const opt = document.createElement("option");
        opt.value = c.id;
        opt.textContent = `${c.remoteAddr} (发送:${c.bytesSent}B 接收:${c.bytesReceived}B)`;
        select.appendChild(opt);
    });
    if (!selectedClientId) {
        els.btnDisconnectClient.disabled = true;
    }
});

// --- Helpers ---
function appendSystem(msg) {
    const now = new Date().toLocaleTimeString();
    els.receivedDisplay.value += `[${now}] [系统] ${msg}\n`;
    if (els.autoScroll.checked) {
        els.receivedDisplay.scrollTop = els.receivedDisplay.scrollHeight;
    }
}

function setStatus(state, text) {
    els.statusBadge.textContent = text || state;
    els.statusBadge.className = "status-" + state;
}

function updateStatusUI(state) {
    switch (state) {
        case "connected":
            setStatus("connected", "已连接");
            els.btnConnect.disabled = true;
            els.btnDisconnect.disabled = false;
            els.btnSend.disabled = false;
            els.btnSendHex.disabled = false;
            els.protocol.disabled = true;
            els.localIp.disabled = true;
            els.localPort.disabled = true;
            // UDP keeps remote fields editable
            if (els.protocol.value !== "udp") {
                els.remoteIp.disabled = true;
                els.remotePort.disabled = true;
            }
            break;
        case "disconnected":
            setStatus("disconnected", "未连接");
            break;
        case "connecting":
            setStatus("connecting", "连接中...");
            break;
        case "error":
            setStatus("error", "错误");
            break;
    }
}

function resetUI() {
    setStatus("disconnected", "未连接");
    els.btnConnect.disabled = false;
    els.btnDisconnect.disabled = true;
    els.btnSend.disabled = true;
    els.btnSendHex.disabled = true;
    els.protocol.disabled = false;
    els.localIp.disabled = false;
    els.localPort.disabled = false;
    els.remoteIp.disabled = false;
    els.remotePort.disabled = false;
    els.clientSelect.innerHTML = '<option value="">-- 选择客户端 --</option>';
    selectedClientId = "";
    els.btnDisconnectClient.disabled = true;

    // Stop auto send
    if (autoSendTimer) {
        clearInterval(autoSendTimer);
        autoSendTimer = null;
    }
    els.autoSend.checked = false;
}

// --- Init: detect local IP on startup ---
(async () => {
    try {
        const ip = await NetService.GetLocalIP();
        if (ip) {
            els.localIp.value = ip;
        }
    } catch (err) {
        console.error("获取本机IP失败:", err);
    }
})();
