## jsonnet-language-server integration for vim

The LSP integration will depend on the vim plugin you're using:

* `mattn/vim-lsp-settings`:
  * Follow new LSP instalation from <https://github.com/mattn/vim-lsp-settings>
  * LSP settings file: [jsonnet-language-server.vim](jsonnet-language-server.vim)
* `neoclide/coc.nvim`:
  * Inside vim, run: `:CocConfig` (to edit `~/.vim/coc-settings.json`)
  * Copy [coc-settings.json](coc-settings.json) content
