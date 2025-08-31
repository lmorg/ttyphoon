{
    # The following code adds mxtty integrations into Zsh

    printf "\033_insert;integration\033\\"
 
    _output_block_begin() {
        printf "\033_begin;output-block;$1\033\\"
    }

    _output_block_end() {
        printf "\033_end;output-block;{\"ExitNum\":$?}\033\\"
    }

    preexec_functions+=(_output_block_begin)
    precmd_functions+=(_output_block_end)
}
