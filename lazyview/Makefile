

tidy:
	go mod tidy

test:
	go test ./...

tag:
ifndef VERSION
	$(error VERSION is not set)
endif
	git tag ${VERSION}

push:
ifndef VERSION
	$(error VERSION is not set)
endif
	git push origin ${VERSION}
