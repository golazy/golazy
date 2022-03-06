

tidy:
	go mod tidy

test:
	go test ./...

release: tag push

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
