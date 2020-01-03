.PHONY: race
race:
	go test -race ./...

.PHONY: coverage
coverage:
	go test -race -coverprofile=coverage.txt -covermode=atomic ./... && go tool cover -html=coverage.txt

.PHONY: autobahn
autobahn:
	docker run -it --rm \
		-v "${PWD}/autobahn/config:/config" \
		-v "${PWD}/autobahn/reports:/reports" \
		--network=host \
		crossbario/autobahn-testsuite \
		/usr/local/bin/wstest \
		--mode fuzzingclient \
		--spec /config/fuzzingclient.json