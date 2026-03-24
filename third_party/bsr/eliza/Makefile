ROOT := ../../..
BIN := /tmp/gripmock-bsr
MODULE := buf.build/connectrpc/eliza
PID := /tmp/gripmock-bsr-eliza.pid

GRPC_PORT ?= 4770
HTTP_PORT ?= 4771

.PHONY: up test down

up:
	go build -o $(BIN) $(ROOT)
	GRPC_PORT=$(GRPC_PORT) HTTP_PORT=$(HTTP_PORT) $(BIN) --stub=. $(MODULE) >/tmp/gripmock-bsr-eliza.log 2>&1 & echo $$! >$(PID)
	sleep 2

test:
	GRPCTESTIFY_ADDRESS=localhost:$(GRPC_PORT) grpctestify .

down:
	kill -INT $$(cat $(PID)) 2>/dev/null || true
	rm -f $(PID)
