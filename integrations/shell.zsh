{
    # The following code adds ttyphoon integrations into Zsh

    #printf "\033_insert;integration\033\\"

    _ttyphoon_capture_active=0

    _output_block_begin() {
        [[ "$_ttyphoon_capture_active" == 1 ]] && return
        _ttyphoon_capture_active=1
        local _ttyphoon_cmdline="$1"
        local _ttyphoon_cmdline_b64
        local _ttyphoon_env_b64
        _ttyphoon_cmdline_b64=$(printf '%s' "$_ttyphoon_cmdline" | command base64 | command tr -d '\n')
        _ttyphoon_env_b64=$(command env -0 | command base64 | command tr -d '\n')
        printf '\033_begin;output-block;{"CmdLineBase64":"%s","EnvVarsBase64":"%s"}\033\\' \
            "$_ttyphoon_cmdline_b64" "$_ttyphoon_env_b64"
        _ttyphoon_capture_active=0
    }

    _output_block_end() {
        printf "\033_end;output-block;{\"ExitNum\":$?}\033\\"
    }

    preexec_functions+=(_output_block_begin)
    precmd_functions+=(_output_block_end)
}
