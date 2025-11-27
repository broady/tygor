# Root Makefile for all examples
MAKEFLAGS += -j

EXAMPLES := newsserver blog protobuf multipackage

.PHONY: all test gen check fmt clean readme precommit $(EXAMPLES)

# Generate precommit targets for each example (enables parallel execution)
define precommit-target
.PHONY: precommit-$(1)
precommit-$(1):
	@echo "=== $(1) ===" && $$(MAKE) -C $(1) test check
endef
$(foreach ex,$(EXAMPLES),$(eval $(call precommit-target,$(ex))))

# Run precommit on all examples in parallel
precommit: $(addprefix precommit-,$(EXAMPLES))
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
