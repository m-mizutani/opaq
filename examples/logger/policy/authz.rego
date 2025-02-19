package authz

allow if {
    input.user == "alice"
    input.action == "read"
}
