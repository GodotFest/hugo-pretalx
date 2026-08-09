.PHONY: build test test-site clean help

# Default target
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the CLI binary
	go build -o hugo-pretalx .
	@echo "Built: ./hugo-pretalx"

test: build test-site ## Run all tests (build + test site)

test-site: ## Build the test Hugo site
	@echo "Building test site..."
	cd test/site && hugo --minify
	@echo ""
	@echo "Verifying output..."
	@test -f test/site/public/index.html && echo "  ✓ Home page"
	@test -f test/site/public/talks/index.html && echo "  ✓ Talks list"
	@test -f test/site/public/speakers/index.html && echo "  ✓ Speakers grid"
	@test -f test/site/public/schedule/index.html && echo "  ✓ Schedule"
	@test -f test/site/public/talks/building-great-software/index.html && echo "  ✓ Talk detail page"
	@test -f test/site/public/speakers/alice-johnson/index.html && echo "  ✓ Speaker detail page"
	@echo ""
	@echo "All tests passed."

clean: ## Remove build artifacts
	rm -f hugo-pretalx
	rm -rf test/site/public
	rm -rf test/site/resources
