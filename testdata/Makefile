.PHONY: deps
deps:
	command -v pup > /dev/null || go install github.com/ericchiang/pup@latest

.index.html:
	curl -s https://x12.org/examples > $@

urls.txt: deps .index.html
	cat .index.html | pup 'a attr{href}' | grep /examples/00 > $@

.PHONY: testdata-x12
testdata-x12: urls.txt
	@./fetch-examples.sh

clean:
	rm *.html
