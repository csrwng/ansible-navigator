# Ansible navigator
  Command-line utility to navigate ansible playbooks and roles

  Usage:
  ```
  ansible-navigator FILENAME ROW COL
  ```

  Result:
  - blank if position does not correspond to a linked file or said file cannot be found
  OR
  - full path to file that is being referenced

  Sample Vim configuration:
  ```vim
  function! AnsibleNavigate()
    let l:cmd="ansible-navigator " . expand('%:p') . " " . line(".") . " " . col(".")
    let l:result=system(l:cmd)
    let l:result_code=v:shell_error
    if l:result_code == 0 && l:result != ""
      execute 'edit' l:result
    endif
  endfunction
  command! AnsibleNavigate call AnsibleNavigate()
  autocmd FileType yaml nnoremap <Localleader>d :AnsibleNavigate<cr>
  ```
