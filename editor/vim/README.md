## jsonnet-language-server integration for vim

The LSP integration will depend on the vim plugin you're using

* `mattn/vim-lsp-settings`:
  * Follow new LSP instalation from <https://github.com/mattn/vim-lsp-settings>
  * LSP settings file: [jsonnet-language-server.vim](jsonnet-language-server.vim)
* `neoclide/coc.nvim`:
  * Inside vim, run: `:CocConfig` (to edit `~/.vim/coc-settings.json`)
  * Copy [coc-settings.json](coc-settings.json) content

Some adjustments you may need to review for above example configs:
* Both are preset to run `jsonnet-language-server -t`, i.e. with
  automatic support for [tanka](https://tanka.dev/) import paths.
* Depending on how you handle `jsonnet` import paths, you may also
  want to add `--jpath <JPATH>` additional search paths for library
  imports.
