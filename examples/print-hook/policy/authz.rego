package authz

allow if {
    print("input", input)
    input.user == "alice"
    input.action == "read"
}
