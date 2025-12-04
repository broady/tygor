# Root Makefile for all examples
# Note: parallelism is inherited from parent make via precommit targets

EXAMPLES := newsserver blog protobuf multipackage reflection typesonly zod react devtools starter-solid with-sqlc

.PHONY: all test gen check fmt clean readme precommit lint-examples $(EXAMPLES)

# Generate precommit targets for each example (enables parallel execution)
# Output is suppressed unless there's a failure
define precommit-target
.PHONY: precommit-$(1)
precommit-$(1):
	@output=$$($$(MAKE) --no-print-directory -C $(1) test-quiet check-quiet 2>&1) || (echo "=== $(1) ==="; echo "$$output"; exit 1)
endef
$(foreach ex,$(EXAMPLES),$(eval $(call precommit-target,$(ex))))

# Run precommit on all examples in parallel
precommit: lint-examples $(addprefix precommit-,$(EXAMPLES))

# Lint: verify all example directories are in EXAMPLES list
# An "example" is a directory with a Makefile that includes common.mk
lint-examples:
	@missing=""; \
	for makefile in */Makefile; do \
		dir=$$(dirname "$$makefile"); \
		if grep -q "common.mk" "$$makefile" 2>/dev/null; then \
			if ! echo " $(EXAMPLES) " | grep -q " $$dir "; then \
				missing="$$missing $$dir"; \
			fi; \
		fi; \
	done; \
	if [ -n "$$missing" ]; then \
		echo "ERROR: Example directories missing from EXAMPLES list:$$missing"; \
		echo "Add them to examples/Makefile"; \
		exit 1; \
	fi

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
check: lint-examples
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
