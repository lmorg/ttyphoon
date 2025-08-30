{
    # The following code adds mxtty integrations into Bash

    printf "\033_insert;integration\033\\"

    _output_block_begin() {
        [ -n "$COMP_LINE" ] && return
        [ "$BASH_COMMAND" = "$PROMPT_COMMAND" ] && return

        printf "\033_begin;output-block;${BASH_COMMAND}\033\\"
    }

    _output_block_end() {
        printf "\033_end;output-block;{\"ExitNum\":$?}\033\\"
    }

    trap '_output_block_begin' DEBUG
    PROMPT_COMMAND='_output_block_end'
}
