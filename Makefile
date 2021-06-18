prepare:
	@rm -rf /tmp/badger
	@go build
	@cd examples/client && go build
	@mkdir /tmp/badger
	@mkdir /tmp/badger/coordinator
	@mkdir -p /tmp/badger/follower1
	@mkdir -p /tmp/badger/follower2

run-example-coordinator:
	@./committer -role=coordinator -nodeaddr=localhost:3000 -followers=localhost:3001,localhost:3002 -committype=two-phase -timeout=1000 -dbpath=/tmp/badger/coordinator -whitelist=127.0.0.1

run-example-follower1:
	@./committer -role=follower -nodeaddr=localhost:3001 -committype=two-phase -timeout=1000 -dbpath=/tmp/badger/follower1 -whitelist=127.0.0.1

run-example-follower2:
	@./committer -role=follower -nodeaddr=localhost:3002 -committype=two-phase -timeout=1000 -dbpath=/tmp/badger/follower2 -whitelist=127.0.0.1

run-example-client:
	@examples/client/client

unit-tests:
	@cd server && go test

functional-tests:
	@go test