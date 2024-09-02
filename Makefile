build:
	go build .

test:
	go test ./...

run: 
	go run . https://wagslane.dev

crawl:
	go run . "https://crawler-test.com/" 3 100