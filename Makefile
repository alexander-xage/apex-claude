PREFIX ?= $(HOME)/.local/bin
CLAUDE_DIR ?= $(or $(CLAUDE_CONFIG_DIR),$(HOME)/.claude)
SKILLS := agent-protocol graphify-discipline git-discipline sign-off apex-init
# Marks a skill directory as installed by apex; install and uninstall never touch one without it.
MARK := .apex-installed

.PHONY: build install uninstall check test

build:
	go build -trimpath -o bin/apex ./cmd/apex

# User skills, not a plugin: plugin skills are always namespaced (docs/design/agents.md).
install: check build
	mkdir -p "$(PREFIX)" "$(CLAUDE_DIR)/skills"
	cp bin/apex "$(PREFIX)/apex"
	for s in $(SKILLS); do d="$(CLAUDE_DIR)/skills/$$s"; \
	  rm -rf "$$d" && cp -R "skills/$$s" "$$d" && touch "$$d/$(MARK)" || exit 1; done

uninstall:
	rm -f "$(PREFIX)/apex"
	for s in $(SKILLS); do d="$(CLAUDE_DIR)/skills/$$s"; \
	  if [ -f "$$d/$(MARK)" ]; then rm -rf "$$d" || exit 1; fi; done

check:
	@command -v graphify >/dev/null || { echo "graphify is not on PATH: uv tool install graphifyy" >&2; exit 1; }
	@grep -q '"ponytail@ponytail"' "$(CLAUDE_DIR)/plugins/installed_plugins.json" 2>/dev/null || \
	  { echo "ponytail plugin is not installed: /plugin marketplace add DietrichGebert/ponytail, then /plugin install ponytail@ponytail" >&2; exit 1; }
	@for s in $(SKILLS); do d="$(CLAUDE_DIR)/skills/$$s"; \
	  if [ -e "$$d" ] && [ ! -f "$$d/$(MARK)" ]; then echo "$$d exists and was not installed by apex; move it first" >&2; exit 1; fi; done

test:
	go test ./...
