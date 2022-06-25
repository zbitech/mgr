compile:
	go build -v ./...

unit_tests:
	go clean -testcache
	export ASSET_PATH_DIRECTORY="${PWD}/../control-plane/zbi/conf" && \
	export DATA_PATH=${PWD}/../fake-zbi/ && \
		go test -v -run Test_CreateIngressAsset ./...
