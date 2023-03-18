TOOLS_BIN=var/bin
export GOBIN=${CURDIR}/$(TOOLS_BIN)

BUF_BIN=$(TOOLS_BIN)/buf

$(BUF_BIN):
	@mkdir -p "$(GOBIN)"
	go install github.com/bufbuild/buf/cmd/buf@v1.15.1

lint: $(BUF_BIN)
	$(BUF_BIN) lint

generate_api: lint $(BUF_BIN)
	$(BUF_BIN) generate --template generated/buf.gen.api.yaml