# Screen Guard Service

Enterprise-oriented real-time screen-share DLP service for detecting sensitive data (PII, secrets, key material) during Zoom/Slack/browser sharing.

## One-Command Run

```bash
python3 -m screen_guard_service
```

This command reads `screen_guard_service/.env` (or process env), applies defaults, and starts the monitor.

## Architecture

### Runtime Layers

1. `__main__.py`
Service bootstrap and single-command launcher.

2. `run_monitor.py`
Main orchestration (manual/auto mode, model options, outputs).

3. `monitor_runtime/`
Modular control-plane components:
- `cli.py`: CLI contract
- `deps.py`: dependency checks
- `alerts.py`: alert routing
- `recording.py`: run metadata and detection artifacts
- `pii_catalog.py`: centralized PII/sensitive rules + profiles

4. Detection engine modules:
- `realtime_monitor.py`: real-time scan loop, dedupe, confirmation, scope filtering, alert invocation
- `fixed_ml_scanner.py`: primary detector (regex + optional NER + adaptive context)
- `modern_ml_scanner.py`: secondary BERT/NER enrichment
- `screen_content_detector.py`: fallback detector

5. Platform integration:
- `platform_integration.py`: share-state detection for Zoom/Slack/etc and auto start/stop logic.

## File Structure

```text
screen_guard_service/
  __init__.py
  __main__.py
  service_config.py
  run_monitor.py
  realtime_monitor.py
  fixed_ml_scanner.py
  modern_ml_scanner.py
  screen_content_detector.py
  platform_integration.py
  requirements.txt
  .env.example
  monitor_runtime/
    __init__.py
    cli.py
    deps.py
    alerts.py
    recording.py
    pii_catalog.py
```

## Detection Coverage

- PII: email, phone, SSN, private IP, card data
- Financial: credit card (including OCR-robust fallback), IBAN, SWIFT/BIC
- Secrets: API keys/tokens/secrets/password exposures
- Crypto/key material:
  - PEM private/public keys
  - PGP private key blocks
  - SSH public keys
  - key assignments in code/config

Profiles:
- `pii_core`: stricter scope, reduced noise
- `pii_full`: broader enterprise coverage (default)
- `all`: no scope filtering

## Enterprise Alerting Behavior

- Severity-aware (`critical/high/medium/low`)
- Human-readable labels and reason text
- Hidden password suppression (masked values ignored)
- Visible password exposure is flagged
- Dedup/cooldown/confirmation to reduce alert floods

## Recorded Artifacts

Per run:
- `monitor_output/runs/<run_id>/runtime.log`
- `monitor_output/runs/<run_id>/detections.jsonl`
- `monitor_output/runs/<run_id>/run_metadata.json`
- `monitor_output/runs/<run_id>/session_report.json` (when available)

## Microservice Integration Path

Recommended deployment model:
- Run this package as a sidecar or local agent process.
- Consume `detections.jsonl` for event streaming into your app/SIEM.
- Optionally route alerts through webhooks/notification channels.
- Wrap `python3 -m screen_guard_service` with your process supervisor:
  - systemd
  - Docker
  - Kubernetes sidecar

## Setup

1. Copy env template:

```bash
cp screen_guard_service/.env.example screen_guard_service/.env
```

2. Install dependencies:

```bash
pip install -r screen_guard_service/requirements.txt
```

3. Run:

```bash
python3 -m screen_guard_service
```

## Common Use Cases

- Live meeting protection for engineering demos
- Secure client support sessions
- Compliance monitoring for remote workforce
- Sensitive-data leakage prevention in screen-share workflows

