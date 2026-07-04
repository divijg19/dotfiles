# Developer Toolchain Architecture

**Problem**: LazyVim language extras install external tooling through Mason by default. This creates duplicate installations and split version authority.

**Decision**: Package ownership is determined by the tool's natural ecosystem. Neovim configures tools; Neovim does not own tools.

### Rules

• Every tool has one owner (Fedora / Go / Cargo).
• Editor runtimes may use Mason (e.g. `lua-language-server`).
• Developer tooling lives outside Mason.

---

### Component Architecture

*   **`packages.lua`** → Executable ownership declarations.
*   **`lsp.lua`** → LSP server ownership / exclusion list.
*   **`package_policy.lua`** → Adapts policies into LazyVim spec overrides.
