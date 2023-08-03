#!/usr/bin/env bash

main() {
    # get names of modules that use ice module into array
    modules=$(lsmod | grep -P '^ice' | grep -Po '\S+$')
    declare -a ice_dependent
    IFS=','
    read -r -a ice_dependent <<<"$modules"

    # declare 2 arrays
    # - first store modules in order of unloading
    # - second stores modules in order of loading
    declare -a mods_to_unload
    declare -a mods_to_load

    mods_to_load+=('ice')
    for i in "${ice_dependent[@]}"; do
        if [[ "$i" =~ [^a-zA-Z] ]]; then
            continue
        fi
        check_module "$i"
    done
    reverse_array mods_to_load mods_to_unload

    # unload and load modules in proper order
    for i in "${mods_to_unload[@]}"; do
        modprobe -r "$i"
    done

    for i in "${mods_to_load[@]}"; do
        modprobe "$i"
    done

    return 0
}

check_module() {
    mod=$1
    mods_to_load+=("$1")
    mods=$(lsmod | grep -P "^$mod" | grep -Po '\S+$')
    if [[ "$mods" =~ [^a-zA-Z] ]]; then
        return 1
    else
        declare -a mod_dependent
        read -r -a mod_dependent <<<"$mods"
        for i in "${mod_dependent[@]}"; do
            check_module "$i"
        done
    fi
}

reverse_array() {
    declare -n arr="$1" rev="$2"
    for i in "${arr[@]}"; do
        rev=("$i" "${rev[@]}")
    done
}

main "$@"
exit
