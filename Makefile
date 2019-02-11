all: build done

build:
	@echo "Building..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w"
build-upx:
	@echo "Building..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w"
	upx holzhacker
clean:
	@echo "Cleanup..."
	@rm holzhacker
done:
	@echo "Done."

