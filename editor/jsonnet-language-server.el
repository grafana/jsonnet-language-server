;;; jsonnet-language-server -- Summary
;; Development lsp registration for Emacs lsp-mode.
;;; Commentary:
;;; Code:
(require 'lsp-mode)

(defcustom lsp-jsonnet-executable "jsonnet-language-server"
  "Command to start the Jsonnet language server."
  :group 'lsp-jsonnet
  :risky t
  :type 'file)
(add-to-list 'lsp-language-id-configuration '(jsonnet-mode . "jsonnet"))
(lsp-register-client
 (make-lsp-client
  :new-connection (lsp-stdio-connection (lambda () lsp-jsonnet-executable))
  :activation-fn (lsp-activate-on "jsonnet")
  :server-id 'jsonnet))
(add-hook 'jsonnet-mode-hook #'lsp-deferred)

(provide 'jsonnet-language-server)
;;; jsonnet-language-server.el ends here
