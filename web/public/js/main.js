/**
 * TangoKore Join Webpage
 *
 * Philosophy: Same SDK, same code, same choices.
 * Just presented differently based on skill level.
 *
 * Choice tree:
 * 1. Skill level: Simple (non-technical) vs Advanced (developer/operator)
 * 2. Machine identity: Auto-generate or provide custom
 * 3. Data exposure: Choose what info to send beyond minimum
 * 4. Credentials: Optional (for pre-provisioned access)
 */

// State
const state = {
    skillLevel: null,
    machineId: null,
    optionalFields: {},
    credentials: null,
};

// UI References
const skillSelection = document.getElementById('skill-selection');
const simpleFlow = document.getElementById('simple-flow');
const advancedFlow = document.getElementById('advanced-flow');
const enrollmentProgress = document.getElementById('enrollment-progress');
const enrollmentSuccess = document.getElementById('enrollment-success');
const enrollmentError = document.getElementById('enrollment-error');

const skillCards = document.querySelectorAll('.skill-card');
const simpleStartBtn = document.getElementById('simple-start-btn');
const advancedStartBtn = document.getElementById('advanced-start-btn');
const backBtn = document.getElementById('back-btn');
const errorBackBtn = document.getElementById('error-back-btn');

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    setupSkillSelection();
    setupSimpleFlow();
    setupAdvancedFlow();
    updatePayloadPreview();
});

// ============================================================================
// SKILL LEVEL SELECTION
// ============================================================================

function setupSkillSelection() {
    skillCards.forEach(card => {
        card.addEventListener('click', () => {
            const skill = card.dataset.skill;
            selectSkillLevel(skill);
        });
    });
}

function selectSkillLevel(skillLevel) {
    state.skillLevel = skillLevel;
    hideAllSections();

    if (skillLevel === 'simple') {
        simpleFlow.style.display = 'block';
        setupSimpleUI();
    } else if (skillLevel === 'advanced') {
        advancedFlow.style.display = 'block';
        setupAdvancedUI();
    }
}

// ============================================================================
// SIMPLE FLOW (Non-Technical Users)
// ============================================================================

function setupSimpleFlow() {
    const hostnameInput = document.getElementById('simple-hostname');
    const agreeCheckbox = document.getElementById('simple-agree');

    // Auto-generate or use custom hostname
    hostnameInput.addEventListener('input', () => {
        updateSimpleMachineId();
    });

    // Enable button when agreed
    agreeCheckbox.addEventListener('change', () => {
        simpleStartBtn.disabled = !agreeCheckbox.checked;
    });

    // Start enrollment
    simpleStartBtn.addEventListener('click', () => {
        buildSimplePayload();
        startEnrollment();
    });
}

function setupSimpleUI() {
    // Show what optional fields are selected
    updateSimpleMachineId();
}

function updateSimpleMachineId() {
    const hostnameInput = document.getElementById('simple-hostname');
    const hostname = hostnameInput.value || generateRandomId();
    const autoIdElement = document.getElementById('simple-auto-id');

    state.machineId = hostname;
    autoIdElement.textContent = hostname;
}

function buildSimplePayload() {
    // Collect optional fields from checkboxes
    state.optionalFields = {
        hostname: document.getElementById('simple-opt-hostname').checked,
        os_version: document.getElementById('simple-opt-os-version').checked,
        cpu_info: document.getElementById('simple-opt-cpu').checked,
        memory_mb: document.getElementById('simple-opt-memory').checked,
        mac_addrs: document.getElementById('simple-opt-mac-addrs').checked,
    };

    console.log('Simple enrollment payload:', {
        issued_id: state.machineId,
        optional: state.optionalFields,
    });
}

// ============================================================================
// ADVANCED FLOW (Technical Users)
// ============================================================================

function setupAdvancedFlow() {
    const issuedIdInput = document.getElementById('advanced-issued-id');

    // Machine ID handling
    issuedIdInput.addEventListener('input', updatePayloadPreview);

    // Optional field checkboxes
    document.querySelectorAll('#advanced-flow input[type="checkbox"]').forEach(checkbox => {
        checkbox.addEventListener('change', updatePayloadPreview);
    });

    // Credentials
    document.getElementById('adv-role-id').addEventListener('input', updatePayloadPreview);
    document.getElementById('adv-secret-id').addEventListener('input', updatePayloadPreview);

    // Start enrollment
    advancedStartBtn.addEventListener('click', () => {
        buildAdvancedPayload();
        startEnrollment();
    });
}

function setupAdvancedUI() {
    // Initialize with empty or auto ID
    document.getElementById('advanced-issued-id').placeholder = `auto: ${generateRandomId()}`;
    updatePayloadPreview();
}

function updatePayloadPreview() {
    if (!advancedFlow || advancedFlow.style.display === 'none') return;

    const issuedId = document.getElementById('advanced-issued-id').value || `[auto-generated]`;
    const optional = {
        hostname: document.getElementById('adv-opt-hostname').checked,
        os_version: document.getElementById('adv-opt-os-version').checked,
        kernel_version: document.getElementById('adv-opt-kernel-version').checked,
        machine_uuid: document.getElementById('adv-opt-machine-uuid').checked,
        cpu_info: document.getElementById('adv-opt-cpu-info').checked,
        memory_mb: document.getElementById('adv-opt-memory').checked,
        mac_addrs: document.getElementById('adv-opt-mac-addrs').checked,
        serial_number: document.getElementById('adv-opt-serial-number').checked,
    };

    const payload = {
        os: 'linux',          // Would be detected
        arch: 'amd64',        // Would be detected
        issued_id: issuedId,
        ...Object.fromEntries(
            Object.entries(optional).filter(([, checked]) => checked).map(([key]) => [key, '...'])
        ),
    };

    if (document.getElementById('adv-role-id').value) {
        payload.role_id = '...';
        payload.secret_id = '...';
    }

    const preview = JSON.stringify(payload, null, 2);
    document.getElementById('payload-preview').textContent = preview;
}

function buildAdvancedPayload() {
    const issuedId = document.getElementById('advanced-issued-id').value;

    state.machineId = issuedId || generateRandomId();

    state.optionalFields = {
        hostname: document.getElementById('adv-opt-hostname').checked,
        os_version: document.getElementById('adv-opt-os-version').checked,
        kernel_version: document.getElementById('adv-opt-kernel-version').checked,
        machine_uuid: document.getElementById('adv-opt-machine-uuid').checked,
        cpu_info: document.getElementById('adv-opt-cpu-info').checked,
        memory_mb: document.getElementById('adv-opt-memory').checked,
        mac_addrs: document.getElementById('adv-opt-mac-addrs').checked,
        serial_number: document.getElementById('adv-opt-serial-number').checked,
    };

    const roleId = document.getElementById('adv-role-id').value;
    const secretId = document.getElementById('adv-secret-id').value;

    if (roleId || secretId) {
        state.credentials = { role_id: roleId, secret_id: secretId };
    }

    console.log('Advanced enrollment payload:', {
        issued_id: state.machineId,
        optional: state.optionalFields,
        credentials: state.credentials ? 'provided' : 'none',
    });
}

// ============================================================================
// ENROLLMENT PROCESS
// ============================================================================

function startEnrollment() {
    hideAllSections();
    enrollmentProgress.style.display = 'block';

    const logs = document.getElementById('logs');
    logs.innerHTML = '';

    addLog('Initializing enrollment...', 'info');
    addLog(`Skill level: ${state.skillLevel}`, 'info');
    addLog(`Machine ID: ${state.machineId}`, 'info');

    // Simulate enrollment flow
    enrollmentFlow();
}

async function enrollmentFlow() {
    try {
        // Step 1: Collect system info
        addLog('Scanning machine information...', 'info');
        updateProgress(20);
        await sleep(500);

        addLog('Operating System: linux', 'info');
        addLog('Architecture: amd64', 'info');
        addLog('Machine ID: ' + state.machineId, 'success');

        // Step 2: Prepare payload
        addLog('Building enrollment payload...', 'info');
        updateProgress(40);
        await sleep(300);

        const selectedFields = Object.entries(state.optionalFields)
            .filter(([_, enabled]) => enabled)
            .map(([field]) => field);

        if (selectedFields.length > 0) {
            addLog(`Optional fields: ${selectedFields.join(', ')}`, 'info');
        } else {
            addLog('No optional fields selected', 'info');
        }

        // Step 3: Send enrollment request
        addLog('Connecting to controller...', 'info');
        updateProgress(60);
        await sleep(500);

        addLog('Sending enrollment request...', 'info');
        updateProgress(70);
        await sleep(800);

        // Step 4: Receive response
        addLog('Verification: fingerprint recognized', 'success');
        addLog('Decision: approved', 'success');
        updateProgress(85);
        await sleep(300);

        addLog('Receiving identity certificate...', 'success');
        updateProgress(95);
        await sleep(200);

        // Step 5: Success
        addLog('Enrollment complete!', 'success');
        updateProgress(100);

        setTimeout(() => {
            showSuccessState();
        }, 500);

    } catch (error) {
        addLog(`Error: ${error.message}`, 'error');
        showErrorState(error);
    }
}

// ============================================================================
// UI HELPERS
// ============================================================================

function hideAllSections() {
    skillSelection.style.display = 'none';
    simpleFlow.style.display = 'none';
    advancedFlow.style.display = 'none';
    enrollmentProgress.style.display = 'none';
    enrollmentSuccess.style.display = 'none';
    enrollmentError.style.display = 'none';
}

function addLog(message, type = 'info') {
    const logs = document.getElementById('logs');
    const entry = document.createElement('div');
    entry.className = `log-entry ${type}`;
    entry.textContent = `[${new Date().toLocaleTimeString()}] ${message}`;
    logs.appendChild(entry);
    logs.scrollTop = logs.scrollHeight;
}

function updateProgress(percent) {
    document.getElementById('progress-fill').style.width = percent + '%';

    const messages = {
        20: 'Scanning machine...',
        40: 'Building payload...',
        60: 'Connecting to controller...',
        70: 'Sending request...',
        85: 'Verifying identity...',
        100: 'Complete!',
    };

    document.getElementById('progress-text').textContent = messages[percent] || 'Processing...';
}

function showSuccessState() {
    hideAllSections();
    enrollmentSuccess.style.display = 'block';

    document.getElementById('identity-id').textContent = state.machineId;
    document.getElementById('identity-status').textContent = 'enrolled';
    document.getElementById('identity-trust').textContent = 'stage-0 (quarantine)';

    backBtn.addEventListener('click', () => {
        resetUI();
    });
}

function showErrorState(error) {
    hideAllSections();
    enrollmentError.style.display = 'block';

    document.getElementById('error-message').textContent = error.message || 'An error occurred';
    document.getElementById('error-details').textContent = error.stack || error.toString();

    errorBackBtn.addEventListener('click', () => {
        resetUI();
    });
}

function resetUI() {
    state.skillLevel = null;
    state.machineId = null;
    state.optionalFields = {};
    state.credentials = null;

    // Clear inputs
    document.getElementById('simple-hostname').value = '';
    document.getElementById('simple-agree').checked = false;
    document.getElementById('advanced-issued-id').value = '';
    document.getElementById('adv-role-id').value = '';
    document.getElementById('adv-secret-id').value = '';

    hideAllSections();
    skillSelection.style.display = 'block';
}

// ============================================================================
// UTILITIES
// ============================================================================

function generateRandomId() {
    return 'machine-' + Math.random().toString(36).substring(2, 10);
}

function sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
}
