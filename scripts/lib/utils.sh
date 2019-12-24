check_git() {
  type git > /dev/null 2>&1 || return 1
}

check_curl() {
  type curl > /dev/null 2>&1 || return 1
}
