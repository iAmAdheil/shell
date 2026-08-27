Type: grilling
Status: resolved

## Question

Does creating or joining a Session require authentication? This reopens Round 1's scope call, which had placed accounts/auth out of scope.

## Answer

Yes — the destination is revised to include a full account layer. Session creation and joining require auth; a User cannot do either without being authenticated. Auth is full accounts (sign-up/login), not a shared passphrase or passwordless-only scheme, so User identity persists across Sessions. Which specific auth method (email+password, OAuth, or both) is still open — see [14-auth-method-and-session-token](14-auth-method-and-session-token.md).
