# Deploy

Systemd unit lives at `sentinel.service` (placeholder — Phase 5).

## Production install (Phase 5+, manual)

```bash
# 1. Build
cd /home/yangwei/full-stack-1
make build

# 2. Install unit
sudo cp deploy/sentinel.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now sentinel

# 3. Tail logs
sudo journalctl -u sentinel -f
```

## Bare-metal dev (current)

Just use `make dev` and `make stop`. No systemd involvement.
