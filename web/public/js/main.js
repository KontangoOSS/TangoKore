/**
 * TangoKore Join Webpage
 * Enrollment UI for machines joining the mesh network
 */

// Configuration
const CONFIG = {
    enrollmentAPI: window.location.origin + '/api',
    sessionTimeout: 300000, // 5 minutes
};

// State
let state = {
    method: 'new',
    sessionToken: null,
    enrollmentData: null,
};

// UI Elements
const methodRadios = document.querySelectorAll('input[name="method"]');
const credentialsSection = document.getElementById('credentials-section');
const enrollBtn = document.getElementById('enroll-btn');
const acceptCheckbox = document.getElementById('accept-disclosure');
const methodSelection = document.getElementById('method-selection');
const enrollmentProgress = document.getElementById('enrollment-progress');
const enrollmentSuccess = document.getElementById('enrollment-success');
const enrollmentError = document.getElementById('enrollment-error');
const logs = document.getElementById('logs');
const retryBtn = document.getElementById('retry-btn');
const startOverBtn = document.getElementById('start-over-btn');

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    setupEventListeners();
    loadOptionalItems();
    checkForReturningMachine();
});

function setupEventListeners() {
    methodRadios.forEach(radio => {
        radio.addEventListener('change', (e) => {
            state.method = e.target.value;
            updateUIForMethod(state.method);
            updateEnrollButtonState();
        });
    });

    acceptCheckbox.addEventListener('change', updateEnrollButtonState);
    enrollBtn.addEventListener('click', startEnrollment);
    retryBtn.addEventListener('click', resetUI);
    startOverBtn.addEventListener('click', resetUI);
}

function updateUIForMethod(method) {
    if (method === 'approle') {
        credentialsSection.style.display = 'block';
    } else {
        credentialsSection.style.display = 'none';
    }
}

function updateEnrollButtonState() {
    const isCheckboxChecked = acceptCheckbox.checked;
    const isCredentialsValid = state.method !== 'approle' ||
        (document.getElementById('role-id').value &&
         document.getElementById('secret-id').value);

    enrollBtn.disabled = !(isCheckboxChecked && isCredentialsValid);
}

function loadOptionalItems() {
    const optionalItems = [
        'Hostname - System name',
        'OS Version - Release number',
        'Kernel Version - Linux/Unix kernel',
        'Machine UUID - BIOS-level identifier',
        'CPU Info - Processor details',
        'Memory - RAM amount',
        'MAC Addresses - Network interfaces',
        'Serial Number - Hardware serial'
    ];

    const itemsHtml = optionalItems
        .map(item => `<li>${item}</li>`)
        .join('');

    document.getElementById('optional-items').innerHTML = `
        <ul style="margin-left: 20px;">
            ${itemsHtml}
        </ul>
    `;
}

function checkForReturningMachine() {
    // Try to detect if this machine was previously enrolled
    // This would be done via device fingerprinting in a real implementation
    // For now, just a placeholder
}

function addLog(message, type = 'info') {
    const entry = document.createElement('div');
    entry.className = `log-entry ${type}`;
    entry.textContent = `[${new Date().toLocaleTimeString()}] ${message}`;
    logs.appendChild(entry);
    logs.scrollTop = logs.scrollHeight;
}

function showSection(section) {
    methodSelection.style.display = 'none';
    enrollmentProgress.style.display = 'none';
    enrollmentSuccess.style.display = 'none';
    enrollmentError.style.display = 'none';

    section.style.display = 'block';
}

function updateProgress(percent) {
    document.getElementById('progress-fill').style.width = percent + '%';
}

async function startEnrollment() {
    showSection(enrollmentProgress);
    logs.innerHTML = '';
    updateProgress(0);

    try {
        addLog('Starting enrollment process...', 'info');
        addLog(`Method: ${state.method}`, 'info');

        // Step 1: Get session token
        addLog('Generating session token...', 'info');
        updateProgress(10);

        const sessionToken = await getSessionToken();
        state.sessionToken = sessionToken;
        addLog(`Session token received: ${sessionToken.substring(0, 20)}...`, 'success');

        // Step 2: Collect machine fingerprint
        addLog('Collecting machine fingerprint...', 'info');
        updateProgress(25);

        const fingerprint = await collectFingerprint();
        addLog(`OS: ${fingerprint.os}`, 'info');
        addLog(`Architecture: ${fingerprint.arch}`, 'info');
        addLog(`Machine ID: ${fingerprint.issued_id}`, 'info');

        // Step 3: Build enrollment payload
        addLog('Building enrollment payload...', 'info');
        updateProgress(40);

        const enrollmentPayload = buildEnrollmentPayload(fingerprint);

        // Step 4: Send enrollment request
        addLog('Sending enrollment request...', 'info');
        updateProgress(55);

        const result = await enrollWithSSE(enrollmentPayload);
        state.enrollmentData = result;

        addLog('Enrollment successful!', 'success');
        updateProgress(100);

        // Step 5: Show success
        setTimeout(() => {
            showSuccessState(result);
        }, 1000);

    } catch (error) {
        addLog(`Error: ${error.message}`, 'error');
        showErrorState(error);
    }
}

async function getSessionToken() {
    // In a real implementation, this would hit /api/session endpoint
    // For now, return a mock token
    return new Promise((resolve) => {
        setTimeout(() => {
            const token = 'sess_' + Math.random().toString(36).substring(2, 15);
            resolve(token);
        }, 500);
    });
}

async function collectFingerprint() {
    return {
        os: getOS(),
        arch: getArch(),
        issued_id: getOrCreateMachineID(),
        hostname: getHostname(),
        os_version: 'unknown', // Would be collected on actual machine
        kernel_version: 'unknown',
        machine_uuid: 'unknown',
        cpu_info: 'unknown',
        memory_mb: 0,
        mac_addrs: [],
        serial_number: 'unknown'
    };
}

function getOS() {
    const ua = navigator.userAgent;
    if (ua.includes('Windows')) return 'windows';
    if (ua.includes('Mac')) return 'darwin';
    if (ua.includes('Linux')) return 'linux';
    return 'unknown';
}

function getArch() {
    // This would be collected from the actual machine
    // Browser can't reliably determine this
    return 'amd64';
}

function getHostname() {
    // Would be collected from the machine
    return 'unknown';
}

function getOrCreateMachineID() {
    // Try to get from localStorage, or generate new
    let id = localStorage.getItem('tangokore_machine_id');
    if (!id) {
        id = 'mid_' + Math.random().toString(36).substring(2, 15);
        localStorage.setItem('tangokore_machine_id', id);
    }
    return id;
}

function buildEnrollmentPayload(fingerprint) {
    const payload = {
        os: fingerprint.os,
        arch: fingerprint.arch,
        issued_id: fingerprint.issued_id,
        session: state.sessionToken,
        method: state.method,
    };

    // Add credentials if approle method
    if (state.method === 'approle') {
        payload.role_id = document.getElementById('role-id').value;
        payload.secret_id = document.getElementById('secret-id').value;
    }

    // Add optional fingerprint data
    Object.assign(payload, {
        hostname: fingerprint.hostname,
        os_version: fingerprint.os_version,
        kernel_version: fingerprint.kernel_version,
        machine_uuid: fingerprint.machine_uuid,
        cpu_info: fingerprint.cpu_info,
        memory_mb: fingerprint.memory_mb,
        mac_addrs: fingerprint.mac_addrs,
        serial_number: fingerprint.serial_number,
    });

    return payload;
}

async function enrollWithSSE(payload) {
    return new Promise((resolve, reject) => {
        const eventSource = new EventSource(CONFIG.enrollmentAPI + '/enroll/stream');
        let identity = null;
        let verified = false;

        eventSource.addEventListener('verify', (event) => {
            const data = JSON.parse(event.data);
            addLog(`Verification: ${data.check} = ${data.passed}`, data.passed ? 'success' : 'warning');
        });

        eventSource.addEventListener('decision', (event) => {
            const data = JSON.parse(event.data);
            addLog(`Decision: ${data.status}`, 'info');
            verified = true;
        });

        eventSource.addEventListener('identity', (event) => {
            const data = JSON.parse(event.data);
            identity = data;
            addLog(`Identity received: ${data.id}`, 'success');
            eventSource.close();
            resolve(data);
        });

        eventSource.addEventListener('error', (event) => {
            eventSource.close();
            reject(new Error('SSE connection failed'));
        });

        // Send enrollment data via POST
        fetch(CONFIG.enrollmentAPI + '/enroll/stream', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(payload)
        }).catch(err => {
            addLog(`POST failed: ${err.message}`, 'error');
        });

        // Timeout
        setTimeout(() => {
            if (!identity) {
                eventSource.close();
                reject(new Error('Enrollment timeout'));
            }
        }, CONFIG.sessionTimeout);
    });
}

function showSuccessState(identity) {
    showSection(enrollmentSuccess);
    updateProgress(100);

    document.getElementById('identity-machine-id').textContent = identity.id || 'unknown';
    document.getElementById('identity-status').textContent = identity.status || 'enrolled';
    document.getElementById('identity-stage').textContent = identity.stage || 'stage-0';

    const method = state.method;
    const platform = getOS();

    // Download command based on platform
    const downloadCmd = generateDownloadCommand(platform);
    document.getElementById('download-cmd-btn').style.display = 'inline-block';
    document.getElementById('download-cmd-btn').onclick = () => {
        navigator.clipboard.writeText(downloadCmd);
        alert('Installation command copied to clipboard!');
    };
}

function generateDownloadCommand(platform) {
    const baseUrl = window.location.origin;
    const sessionToken = state.sessionToken;
    const method = state.method;

    let cmd = `curl -k ${baseUrl}/install`;

    if (method === 'approle') {
        cmd += ` --role-id ${document.getElementById('role-id').value}`;
        cmd += ` --secret-id ${document.getElementById('secret-id').value}`;
    } else if (method === 'scan') {
        cmd += ` --scan`;
    }

    cmd += ` | bash`;

    return cmd;
}

function showErrorState(error) {
    showSection(enrollmentError);
    document.getElementById('error-message').textContent = error.message;
    document.getElementById('error-details').textContent = error.stack || error.toString();
}

function resetUI() {
    state = {
        method: 'new',
        sessionToken: null,
        enrollmentData: null,
    };

    document.getElementById('role-id').value = '';
    document.getElementById('secret-id').value = '';
    document.getElementById('accept-disclosure').checked = false;

    // Reset method selection
    document.querySelector('input[name="method"][value="new"]').checked = true;
    updateUIForMethod('new');
    updateEnrollButtonState();

    showSection(methodSelection);
    logs.innerHTML = '';
    updateProgress(0);
}

// Handle visibility changes (pause/resume SSE if tab hidden)
document.addEventListener('visibilitychange', () => {
    if (document.hidden) {
        addLog('Tab hidden, enrollment may pause...', 'warning');
    } else {
        addLog('Tab visible again', 'info');
    }
});
