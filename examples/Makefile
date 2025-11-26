# Root Makefile for all examples

EXAMPLES := newsserver blog protobuf

.PHONY: all test gen check fmt clean precommit $(EXAMPLES)

# Run precommit on all examples
precommit:
	@echo "Running precommit checks on all examples..."
	@for ex in $(EXAMPLES); do \
		echo "=== $$ex ==="; \
		$(MAKE) -C $$ex gen test check || exit 1; \
	done
	@echo "All examples OK"

# Run tests on all examples
test:
	@for ex in $(EXAMPLES); do \
		echo "=== $$ex ==="; \
		$(MAKE) -C $$ex test || exit 1; \
	done

# Generate types for all examples
gen:
	@for ex in $(EXAMPLES); do \
		$(MAKE) -C $$ex gen || exit 1; \
	done

# Check all examples
check:
	@for ex in $(EXAMPLES); do \
		$(MAKE) -C $$ex check || exit 1; \
	done

# Format all examples
fmt:
	@for ex in $(EXAMPLES); do \
		$(MAKE) -C $$ex fmt || exit 1; \
	done

# Clean all examples
clean:
	@for ex in $(EXAMPLES); do \
		$(MAKE) -C $$ex clean || exit 1; \
	done
