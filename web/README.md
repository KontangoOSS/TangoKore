# TangoKore Join Webpage

The join webpage is the user-facing enrollment interface. It provides:

- **Method Selection**: Choose how to enroll (new machine, returning machine, pre-provisioned)
- **Data Transparency**: Show exactly what data will be sent and why
- **Real-time Logs**: Display enrollment progress with live logs
- **Multi-platform Support**: Works on Linux, macOS, Windows

## Running Locally

```bash
# Simple HTTP server (Python 3)
cd web/public
python3 -m http.server 8000

# Or with Node.js
npx http-server public -p 8000

# Then open: http://localhost:8000
```

## Structure

```
web/
├── public/           # Static files served directly
│   ├── index.html    # Main enrollment interface
│   ├── css/
│   │   └── style.css # Styling
│   └── js/
│       └── main.js   # Enrollment logic
└── README.md         # This file
```

## Integration

The webpage expects these API endpoints to exist:

- `POST /api/enroll/stream` - SSE enrollment endpoint
- `GET /api/session` - Generate session token (optional)

The page handles:
- **New Machines**: Send OS + arch + machine ID
- **Returning Machines**: Use `--scan` flag to match by fingerprint
- **Pre-Provisioned**: Send AppRole credentials (role_id + secret_id)

## Disclosure

The page shows what data will be sent:

**Required (always sent)**:
- Operating System (OS detection)
- Architecture (CPU type)
- Machine ID (unique identifier)

**Optional (enhanced fingerprinting)**:
- Hostname, OS version, kernel version
- CPU info, memory, MAC addresses
- Serial number, UUID

Users can review and accept before enrollment starts.

## Customization

The UI can be customized:

- **Colors**: Edit CSS variables in `style.css` (`:root`)
- **Branding**: Replace header content in `index.html`
- **Methods**: Add new enrollment methods by extending JavaScript
- **Themes**: Add dark mode, custom fonts, etc.

## Security Notes

- The page communicates via HTTPS (enforced on production)
- Session tokens are short-lived
- Credentials (role_id, secret_id) are only sent to `/api/enroll/stream`
- Machine ID is generated client-side and stored locally
- No analytics or tracking (100% user privacy)

## Error Handling

The page gracefully handles:
- Network failures (shows error with details)
- Timeouts (5-minute default, configurable)
- Tab visibility (pauses/resumes SSE on tab change)
- API errors (displays error details from server)

## Testing

```bash
# Local testing
cd web/public
python3 -m http.server 8000

# Test enrollment flow
# 1. Select method (new/scan/approle)
# 2. Review data disclosure
# 3. Click "Start Enrollment"
# 4. Watch logs for verify → decision → identity events
```

## Future Enhancements

- [ ] Dark mode toggle
- [ ] Language selection
- [ ] QR code for CLI integration
- [ ] Advanced fingerprinting details
- [ ] Offline mode (service workers)
- [ ] Installation status polling
- [ ] Certificate export/download

---

**Philosophy**: The page should be understandable by anyone who just learned what the terminal is. No jargon. Clear next steps.
