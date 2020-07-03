.DEFAULT_GOAL := help

test: test_go test_sceanrio

setup:
build:
clean:

test_go:
	go test templates_test.go -v

test_sceanrio:
	./template_test_scenario.sh

help:
	@echo "EDEN is the harness for testing EVE and ADAM"
	@echo
	@echo "This Makefile automates commons tasks of EDEN testing"
	@echo
	@echo "Commonly used maintenance and development targets:"
	@echo "   test          run tests"
	@echo
	@echo "You need install requirements for EVE (look at https://github.com/lf-edge/eve#install-dependencies)."
	@echo "You need access to docker socket and installed qemu packages."

