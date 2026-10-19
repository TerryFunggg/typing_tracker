;;; typing-tracker.el --- Send local coding heartbeats to tt serve -*- lexical-binding: t -*-

;; Loaded from ~/.emacs.d/init.el. Remove that load line to stop.

(require 'json)
(require 'subr-x)
(require 'url)

(defvar typing-tracker--last (make-hash-table :test 'equal))
(defvar typing-tracker--timer nil)
(defvar typing-tracker--warned nil)
(defvar typing-tracker--enabled nil)

(defun typing-tracker--dir ()
  (expand-file-name "typing-tracker"
                    (or (getenv "XDG_CONFIG_HOME")
                        (expand-file-name "~/.config"))))

(defun typing-tracker--warn (msg)
  (unless typing-tracker--warned
    (setq typing-tracker--warned t)
    (message "typing-tracker: %s" msg)))

(defun typing-tracker--port ()
  (let ((file (expand-file-name "config.toml" (typing-tracker--dir)))
        (port 57147))
    (when (file-readable-p file)
      (with-temp-buffer
        (insert-file-contents file)
        (goto-char (point-min))
        (while (not (eobp))
          (let ((line (string-trim (buffer-substring-no-properties
                                    (line-beginning-position)
                                    (line-end-position)))))
            (when (and (not (string-prefix-p "#" line))
                       (string-match "\\`port[[:space:]]*=[[:space:]]*\\([0-9]+\\)" line))
              (setq port (string-to-number (match-string 1 line)))))
          (forward-line 1))))
    (if (> port 0) port 57147)))

(defun typing-tracker--token ()
  (let ((file (expand-file-name "token" (typing-tracker--dir))))
    (when (file-readable-p file)
      (string-trim (with-temp-buffer
                     (insert-file-contents file)
                     (buffer-string))))))

(defun typing-tracker--language ()
  (symbol-name major-mode))

(defun typing-tracker--due (path)
  (let ((prev (gethash path typing-tracker--last)))
    (or (null prev) (> (- (float-time) prev) 30))))

(defun typing-tracker--done (status)
  (if (or (plist-get status :error)
          (not (and (goto-char (point-min))
                    (re-search-forward "^HTTP/.* \\([0-9]+\\)" nil t)
                    (string-prefix-p "2" (match-string 1)))))
      (typing-tracker--warn "server did not accept the heartbeat (is tt serve running?)"))
  (kill-buffer (current-buffer)))

(defun typing-tracker--send (path language is-write)
  (cond
   ((or (not (stringp path))
        (not (file-name-absolute-p path))
        (file-directory-p path))
    nil)
   ((not (typing-tracker--token))
    (typing-tracker--warn "no token yet; run tt serve once"))
   (t
    (puthash path (float-time) typing-tracker--last)
    (let* ((url-request-method "POST")
           (url-request-extra-headers
            `(("Content-Type" . "application/json")
              ("Authorization" . ,(concat "Bearer " (typing-tracker--token)))))
           (url-request-data
            (encode-coding-string
             (json-encode `((entity . ,path)
                            (editor . "emacs")
                            (language . ,language)
                            (is_write . ,(if is-write t json-false))
                            (time . ,(float-time))))
             'utf-8))
           (url-proxy-services nil)
           (url (format "http://127.0.0.1:%d/heartbeat" (typing-tracker--port))))
      (condition-case nil
          (url-retrieve url #'typing-tracker--done nil t t)
        (error (typing-tracker--warn "could not reach tt serve")))))))

(defun typing-tracker--trackable ()
  (let ((path (buffer-file-name)))
    (and (stringp path)
         (file-name-absolute-p path)
         (not (file-directory-p path))
         path)))

(defun typing-tracker--opened ()
  (let ((path (typing-tracker--trackable)))
    (when path
      (typing-tracker--send path (typing-tracker--language) nil))))

(defun typing-tracker--saving ()
  (let ((path (typing-tracker--trackable)))
    (when path
      (typing-tracker--send path (typing-tracker--language) t))))

(defun typing-tracker--changed (&rest _)
  (let ((path (typing-tracker--trackable)))
    (when (and path (typing-tracker--due path) (not (timerp typing-tracker--timer)))
      (let ((language (typing-tracker--language)))
        (setq typing-tracker--timer
              (run-with-timer
               1 nil
               (lambda ()
                 (setq typing-tracker--timer nil)
                 (when (typing-tracker--due path)
                   (typing-tracker--send path language nil)))))))))

(defun typing-tracker-enable ()
  "Start sending heartbeats for file buffers."
  (unless typing-tracker--enabled
    (setq typing-tracker--enabled t)
    (add-hook 'find-file-hook #'typing-tracker--opened)
    (add-hook 'before-save-hook #'typing-tracker--saving)
    (add-hook 'after-change-functions #'typing-tracker--changed)))

(typing-tracker-enable)

(provide 'typing-tracker)
;;; typing-tracker.el ends here
