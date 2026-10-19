" Send local coding heartbeats to tt serve.
" Loaded from ~/.vimrc. Remove that source line to stop.

if exists('g:loaded_typing_tracker')
  finish
endif
let g:loaded_typing_tracker = 1

let s:last = {}
let s:warned = 0

function! s:ConfigDir() abort
  if !empty($XDG_CONFIG_HOME)
    return $XDG_CONFIG_HOME . '/typing-tracker'
  endif
  return expand('~/.config/typing-tracker')
endfunction

function! s:Warn(msg) abort
  if s:warned
    return
  endif
  let s:warned = 1
  echom 'typing-tracker: ' . a:msg
endfunction

function! s:Port() abort
  let path = s:ConfigDir() . '/config.toml'
  if !filereadable(path)
    return 57147
  endif
  for line in readfile(path)
    let line = trim(line)
    if line ==# '' || line[0] ==# '#'
      continue
    endif
    if line =~# '^port\s*='
      let port = str2nr(matchstr(line, '\d\+'))
      if port > 0
        return port
      endif
    endif
  endfor
  return 57147
endfunction

function! s:Token() abort
  let path = s:ConfigDir() . '/token'
  if !filereadable(path)
    return ''
  endif
  return trim(join(readfile(path, '', 1), ''))
endfunction

function! s:OnExit(job, status) abort
  if a:status != 0
    call s:Warn('server did not accept the heartbeat (is tt serve running?)')
  endif
endfunction

function! s:Post(path, is_write) abort
  if !executable('curl')
    call s:Warn('curl is not installed')
    return
  endif
  let token = s:Token()
  if token ==# ''
    call s:Warn('no token yet; run tt serve once')
    return
  endif
  let payload = json_encode({
        \ 'entity': a:path,
        \ 'editor': 'vim',
        \ 'language': &filetype,
        \ 'is_write': a:is_write ? v:true : v:false,
        \ 'time': localtime(),
        \ })
  let header = 'header = "Authorization: Bearer ' . token . "\"\n"
  let url = printf('http://127.0.0.1:%d/heartbeat', s:Port())
  let job = job_start(['curl', '-sS', '-f', '-m', '2', '-o', '/dev/null',
        \ '-X', 'POST', '-H', 'Content-Type: application/json',
        \ '-K', '-', '-d', payload, url], #{
        \ in_mode: 'raw',
        \ out_io: 'null',
        \ err_io: 'null',
        \ exit_cb: function('s:OnExit'),
        \ })
  if job_status(job) ==# 'fail'
    call s:Warn('could not start curl')
    return
  endif
  call ch_sendraw(job, header)
  call ch_close_in(job)
endfunction

function! s:Maybe(is_write) abort
  if &buftype !=# ''
    return
  endif
  let path = expand('%:p')
  if path ==# '' || isdirectory(path)
    return
  endif
  let now = localtime()
  if !a:is_write && has_key(s:last, path) && now - s:last[path] < 30
    return
  endif
  let s:last[path] = now
  call s:Post(path, a:is_write)
endfunction

augroup typing_tracker
  autocmd!
  autocmd BufEnter,CursorHold,TextChanged,TextChangedI * call s:Maybe(0)
  autocmd BufWritePost * call s:Maybe(1)
augroup END
