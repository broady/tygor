# Root Makefile for all examples

EXAMPLES := newsserver blog protobuf

.PHONY: all test gen check fmt clean readme precommit $(EXAMPLES)

# Run precommit on all examples (check runs gen and readme)
precommit:
	@echo "Running precommit checks on all examples..."
	@for ex in $(EXAMPLES); do \
		echo "=== $$ex ==="; \
		$(MAKE) -C $$ex test check || exit 1; \
	done
	@echo "All examples OK"

# Update all READMEs
readme:
	@for ex in $(EXAMPLES); do \
		$(MAKE) -C $$ex readme || exit 1; \
	done

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
