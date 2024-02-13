.PHONY: build preprocess clean install install-release



preprocess: 
	echo "using makefile"
	@echo "Preprocessing..."
	go build ./cmd/ybbuilder
	./ybbuilder 
	@-rm -f ybbuilder
	@-del  .\ybbuilder.exe
	@echo "Done."



build:preprocess
	@echo "Building..."
	go build ./cmd/ybcli
	go build ./cmd/yb
	@echo "Done."

clean:
	@echo "Cleaning..."
	@-rm -f yb
	@-rm -f ybbuilder
	@-rm -f ybcli
	@-del  .\yb.exe
	@-del  .\ybbuilder.exe
	@-del  .\ybcli.exe
	@echo "Done."

install:preprocess
	@echo "Installing..."
	go install ./cmd/yb
	go install ./cmd/ybcli
	go install ./cmd/ybbuilder
	@echo "Done."

install-release:preprocess
	@echo "Installing..."
	go install -ldflags "-s -w" -tags=release ./cmd/yb
	go install -ldflags "-s -w" -tags=release ./cmd/ybcli
	go install -ldflags "-s -w" -tags=release ./cmd/ybbuilder
	@echo "Done."