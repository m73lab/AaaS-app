# AaaS deployment helper. Replace HOMELAB02 with your host.
#
# Full server-side stack (recommended, most professional):
#   export HOMELAB02=user@homelab02
#   export REDIS_PASSWORD='...'
#   export AaaS_MASTER_KEY='...'          # 32-byte envelope-encryption master
#   make deploy                           # certs + render + ship + docker up
#   make tunnel                           # ssh -L 8080:localhost:8080
#   # point your LLM client at http://localhost:8080/v1/chat/completions
#
# Local development (no Docker):
#   make run                             # go run with config.yaml (memory vault)
#
# Stop:
#   make stop                            # docker compose down on homelab02

HOMELAB02 ?= user@homelab02
export HOMELAB02

.PHONY: build gen-certs render deploy tunnel run stop

build:
	go build -o bin/aaas-proxy ./cmd/proxy

gen-certs:
	bash scripts/gen-certs.sh

render: gen-certs
	bash scripts/render-config.sh

deploy: render
	bash scripts/setup-homelab02.sh

tunnel:
	bash scripts/tunnel.sh

run:
	go run ./cmd/proxy -config config.yaml

stop:
	ssh $(HOMELAB02) 'cd /opt/aaas && docker compose down'
