Clone all LSP reference sources into `.scratch/` for oracle verification during development.

## Steps

1. Create `.scratch/` directory if it doesn't exist
2. Clone the following repos with `--depth 1` (shallow) in parallel:
   - `https://github.com/golang/tools.git` → `.scratch/gopls/` (contains gopls)
   - `https://github.com/typescript-language-server/typescript-language-server.git` → `.scratch/typescript-language-server/`
   - `https://github.com/microsoft/pyright.git` → `.scratch/pyright/`
   - `https://github.com/rust-lang/rust-analyzer.git` → `.scratch/rust-analyzer/`
   - `https://github.com/eclipse-jdtls/eclipse.jdt.ls.git` → `.scratch/eclipse.jdt.ls/`
   - `https://github.com/phpactor/phpactor.git` → `.scratch/phpactor/`
   - `https://github.com/castwide/solargraph.git` → `.scratch/solargraph/`
3. For clangd, use a sparse checkout of the LLVM monorepo to avoid cloning the entire project:
   - `git clone --depth 1 --filter=blob:none --sparse https://github.com/llvm/llvm-project.git .scratch/clangd`
   - Then `git sparse-checkout set clang-tools-extra/clangd`
4. Skip any repos that already exist in `.scratch/`
5. Ensure `.scratch/` is in `.gitignore`
6. Report which repos were cloned and which were skipped
